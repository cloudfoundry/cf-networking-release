package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"lib/common"
	"lib/nonmutualtls"
	"lib/poller"

	"policy-server/adapter"
	"policy-server/api"
	"policy-server/api/api_v0"
	"policy-server/cc_client"
	"policy-server/cleaner"
	"policy-server/config"
	"policy-server/handlers"
	psmiddleware "policy-server/middleware"
	"policy-server/store"
	"policy-server/uaa_client"

	"policy-server/db"

	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	middlewareAdapter "code.cloudfoundry.org/cf-networking-helpers/middleware/adapter"
	"code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"
	"github.com/cloudfoundry/dropsonde"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
	"github.com/tedsuo/rata"
)

const (
	jobPrefix       = "policy-server"
	dropsondeOrigin = "policy-server"
)

var (
	logPrefix = "cfnetworking"
)

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	conf, err := config.New(*configFilePath)
	if err != nil {
		log.Fatalf("%s.%s: could not read config file: %s", logPrefix, jobPrefix, err)
	}

	if conf.LogPrefix != "" {
		logPrefix = conf.LogPrefix
	}

	logger, reconfigurableSink := lagerflags.NewFromConfig(fmt.Sprintf("%s.%s", logPrefix, jobPrefix), common.GetLagerConfig())

	var tlsConfig *tls.Config
	if conf.SkipSSLValidation {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: conf.SkipSSLValidation,
		}
	} else {
		tlsConfig, err = nonmutualtls.NewClientTLSConfig(conf.UAACA, conf.CCCA)
		if err != nil {
			log.Fatalf("%s.%s error creating tls config: %s", logPrefix, jobPrefix, err) // not tested
		}
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	uaaClient := &uaa_client.Client{
		BaseURL:    fmt.Sprintf("%s:%d", conf.UAAURL, conf.UAAPort),
		Name:       conf.UAAClient,
		Secret:     conf.UAAClientSecret,
		HTTPClient: httpClient,
		Logger:     logger,
	}

	whoamiHandler := &handlers.WhoAmIHandler{
		Marshaler: marshal.MarshalFunc(json.Marshal),
	}

	uptimeHandler := &handlers.UptimeHandler{
		StartTime: time.Now(),
	}

	storeGroup := &store.GroupTable{}
	destination := &store.DestinationTable{}
	policy := &store.PolicyTable{}

	logger.Info("getting db connection", lager.Data{})
	connectionPool := db.NewConnectionPool(
		conf.Database,
		conf.MaxOpenConnections,
		conf.MaxIdleConnections,
		logPrefix,
		jobPrefix,
		logger,
	)
	logger.Info("db connection retrieved", lager.Data{})

	egressDataStore := &store.EgressPolicyStore{
		EgressPolicyRepo: &store.EgressPolicyTable{
			Conn: connectionPool,
		},
		TerminalsRepo: &store.TerminalsTable{},
	}

	dataStore := store.New(
		connectionPool,
		storeGroup,
		destination,
		policy,
		conf.TagLength,
	)

	if err != nil {
		log.Fatalf("%s.%s: failed to construct datastore: %s", logPrefix, jobPrefix, err) // not tested
	}

	tagDataStore := store.NewTagStore(connectionPool, &store.GroupTable{}, conf.TagLength)

	metricsSender := &metrics.MetricsSender{
		Logger: logger.Session("time-metric-emitter"),
	}

	wrappedStore := &store.MetricsWrapper{
		Store:         dataStore,
		TagStore:      tagDataStore,
		MetricsSender: metricsSender,
	}

	policyCollectionStore := &store.PolicyCollectionStore{
		Conn:              connectionPool,
		PolicyStore:       wrappedStore,
		EgressPolicyStore: egressDataStore,
	}

	wrappedPolicyCollectionStore := &store.PolicyCollectionMetricsWrapper{
		Store:         policyCollectionStore,
		MetricsSender: metricsSender,
	}

	errorResponse := &httperror.ErrorResponse{
		MetricsSender: metricsSender,
	}

	ccClient := &cc_client.Client{
		JSONClient: json_client.New(logger.Session("cc-json-client"), httpClient, conf.CCURL),
		Logger:     logger,
	}

	policyGuard := handlers.NewPolicyGuard(uaaClient, ccClient)
	quotaGuard := handlers.NewQuotaGuard(wrappedStore, conf.MaxPolicies)
	policyFilter := handlers.NewPolicyFilter(uaaClient, ccClient, 100)

	egressValidator := &api.EgressValidator{
		UAAClient: uaaClient,
		CCClient:  ccClient,
	}

	payloadValidator := &api.PayloadValidator{PolicyValidator: &api.Validator{}, EgressPolicyValidator: egressValidator}
	policyMapperV0 := api_v0.NewMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal), &api_v0.Validator{})
	policyMapperV1 := api.NewMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal), payloadValidator)

	createPolicyHandlerV1 := handlers.NewPoliciesCreate(wrappedPolicyCollectionStore, policyMapperV1,
		policyGuard, quotaGuard, errorResponse)
	createPolicyHandlerV0 := handlers.NewPoliciesCreate(wrappedPolicyCollectionStore, policyMapperV0,
		policyGuard, quotaGuard, errorResponse)

	deletePolicyHandlerV1 := handlers.NewPoliciesDelete(wrappedPolicyCollectionStore, policyMapperV1,
		policyGuard, errorResponse)
	deletePolicyHandlerV0 := handlers.NewPoliciesDelete(wrappedPolicyCollectionStore, policyMapperV0,
		policyGuard, errorResponse)

	policiesIndexHandlerV1 := handlers.NewPoliciesIndex(wrappedStore, egressDataStore, policyMapperV1, policyFilter, errorResponse)
	policiesIndexHandlerV0 := handlers.NewPoliciesIndex(wrappedStore, egressDataStore, policyMapperV0, policyFilter, errorResponse)

	egressDestinationMapper := &api.EgressDestinationMapper{
		Marshaler: marshal.MarshalFunc(json.Marshal),
	}

	egressDestinationStore := &store.EgressDestinationStore{
		Conn:                    connectionPool,
		EgressDestinationRepo:   &store.EgressDestinationTable{},
		TerminalsRepo:           &store.TerminalsTable{},
		DestinationMetadataRepo: &store.DestinationMetadataTable{},
	}

	destinationsIndexHandlerV1 := &handlers.DestinationsIndex{
		ErrorResponse:           errorResponse,
		EgressDestinationStore:  egressDestinationStore,
		EgressDestinationMapper: egressDestinationMapper,
		Logger:                  logger,
	}

	createDestinationsHandlerV1 := &handlers.DestinationsCreate{
		ErrorResponse:           errorResponse,
		EgressDestinationStore:  egressDestinationStore,
		EgressDestinationMapper: egressDestinationMapper,
		Logger:                  logger,
	}

	policyCleaner := cleaner.NewPolicyCleaner(logger.Session("policy-cleaner"), wrappedPolicyCollectionStore, uaaClient,
		ccClient, 100, time.Duration(5)*time.Second)

	policiesCleanupHandler := handlers.NewPoliciesCleanup(policyMapperV1, policyCleaner, errorResponse)

	tagsIndexHandler := handlers.NewTagsIndex(wrappedStore, marshal.MarshalFunc(json.Marshal), errorResponse)

	healthHandler := handlers.NewHealth(wrappedStore, errorResponse)

	checkVersionWrapper := &handlers.CheckVersionWrapper{
		ErrorResponse: errorResponse,
		RataAdapter:   adapter.RataAdapter{},
	}

	metricsWrap := func(name string, handler http.Handler) http.Handler {
		metricsWrapper := middleware.MetricWrapper{
			Name:          name,
			MetricsSender: metricsSender,
		}
		return metricsWrapper.Wrap(handler)
	}

	logWrapper := middleware.LogWrapper{
		UUIDGenerator: &middlewareAdapter.UUIDAdapter{},
	}

	logWrap := func(handler http.Handler) http.Handler {
		return logWrapper.LogWrap(logger, handler)
	}

	versionWrap := func(v1Handler, v0Handler http.Handler) http.Handler {
		return checkVersionWrapper.CheckVersion(map[string]http.Handler{
			"v1": v1Handler,
			"v0": v0Handler,
		})
	}

	authAdminWrap := func(handler http.Handler) http.Handler {
		networkAdminAuthenticator := handlers.Authenticator{
			Client:        uaaClient,
			Scopes:        []string{"network.admin"},
			ErrorResponse: errorResponse,
			ScopeChecking: true,
		}
		return networkAdminAuthenticator.Wrap(handler)
	}

	authWriteWrap := func(handler http.Handler) http.Handler {
		networkWriteAuthenticator := handlers.Authenticator{
			Client:        uaaClient,
			Scopes:        []string{"network.admin", "network.write"},
			ErrorResponse: errorResponse,
			ScopeChecking: !conf.EnableSpaceDeveloperSelfService,
		}
		return networkWriteAuthenticator.Wrap(handler)
	}

	externalRoutes := rata.Routes{
		{Name: "uptime", Method: "GET", Path: "/"},
		{Name: "uptime", Method: "GET", Path: "/networking"},
		{Name: "health", Method: "GET", Path: "/health"},
		{Name: "whoami", Method: "GET", Path: "/networking/:version/external/whoami"},
		{Name: "create_policies", Method: "POST", Path: "/networking/:version/external/policies"},
		{Name: "delete_policies", Method: "POST", Path: "/networking/:version/external/policies/delete"},
		{Name: "policies_index", Method: "GET", Path: "/networking/:version/external/policies"},
		{Name: "destinations_index", Method: "GET", Path: "/networking/:version/external/destinations"},
		{Name: "destinations_create", Method: "POST", Path: "/networking/:version/external/destinations"},
		{Name: "cleanup", Method: "POST", Path: "/networking/:version/external/policies/cleanup"},
		{Name: "tags_index", Method: "GET", Path: "/networking/:version/external/tags"},
	}

	corsMiddleware := psmiddleware.CORS{}
	externalRoutesWithOptions := corsMiddleware.AddOptionsRoutes("options", externalRoutes)

	corsOptionsWrapper := func(handler http.Handler) http.Handler {
		wrapper := handlers.CORSOptionsWrapper{
			RataRoutes:         externalRoutesWithOptions,
			AllowedCORSDomains: conf.AllowedCORSDomains,
		}
		return wrapper.Wrap(handler)
	}

	externalHandlers := rata.Handlers{
		"options": corsOptionsWrapper(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		})),
		"uptime": corsOptionsWrapper(metricsWrap("Uptime", logWrap(uptimeHandler))),
		"health": corsOptionsWrapper(metricsWrap("Health", logWrap(healthHandler))),

		"create_policies": corsOptionsWrapper(metricsWrap("CreatePolicies",
			logWrap(versionWrap(authWriteWrap(createPolicyHandlerV1), authWriteWrap(createPolicyHandlerV0))))),

		"delete_policies": corsOptionsWrapper(metricsWrap("DeletePolicies",
			logWrap(versionWrap(authWriteWrap(deletePolicyHandlerV1), authWriteWrap(deletePolicyHandlerV0))))),

		"policies_index": corsOptionsWrapper(metricsWrap("PoliciesIndex",
			logWrap(versionWrap(authWriteWrap(policiesIndexHandlerV1), authWriteWrap(policiesIndexHandlerV0))))),

		"destinations_index": corsOptionsWrapper(metricsWrap("DestinationsIndex",
			logWrap(versionWrap(authAdminWrap(destinationsIndexHandlerV1), authAdminWrap(destinationsIndexHandlerV1))))),

		"destinations_create": corsOptionsWrapper(metricsWrap("DestinationsCreate",
			logWrap(authAdminWrap(createDestinationsHandlerV1)))),

		"cleanup": corsOptionsWrapper(metricsWrap("Cleanup",
			logWrap(versionWrap(authAdminWrap(policiesCleanupHandler), authAdminWrap(policiesCleanupHandler))))),

		"tags_index": corsOptionsWrapper(metricsWrap("TagsIndex",
			logWrap(versionWrap(authAdminWrap(tagsIndexHandler), authAdminWrap(tagsIndexHandler))))),

		"whoami": corsOptionsWrapper(metricsWrap("WhoAmI",
			logWrap(versionWrap(authAdminWrap(whoamiHandler), authAdminWrap(whoamiHandler))))),
	}

	err = dropsonde.Initialize(conf.MetronAddress, dropsondeOrigin)
	if err != nil {
		log.Fatalf("%s.%s: initializing dropsonde: %s", logPrefix, jobPrefix, err)
	}

	metricsEmitter := common.InitMetricsEmitter(logger, wrappedStore)
	externalServer := common.InitServer(logger, nil, conf.ListenHost, conf.ListenPort, externalHandlers, externalRoutesWithOptions)
	poller := initPoller(logger, conf, policyCleaner)
	debugServer := debugserver.Runner(fmt.Sprintf("%s:%d", conf.DebugServerHost, conf.DebugServerPort), reconfigurableSink)

	members := grouper.Members{
		{"metrics_emitter", metricsEmitter},
		{"http_server", externalServer},
		{"policy-cleaner-poller", poller},
		{"debug-server", debugServer},
	}

	logger.Info("starting external server", lager.Data{"listen-address": conf.ListenHost, "port": conf.ListenPort})

	group := grouper.NewOrdered(os.Interrupt, members)
	monitor := ifrit.Invoke(sigmon.New(group))

	err = <-monitor.Wait()
	if connectionPool != nil {
		connectionPool.Close()
	}
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}

	logger.Info("exited")
}

func initPoller(logger lager.Logger, conf *config.Config, policyCleaner *cleaner.PolicyCleaner) ifrit.Runner {
	pollInterval := time.Duration(conf.CleanupInterval) * time.Second

	return &poller.Poller{
		Logger:          logger.Session("policy-cleaner-poller"),
		PollInterval:    pollInterval,
		SingleCycleFunc: policyCleaner.DeleteStalePoliciesWrapper,
	}
}
