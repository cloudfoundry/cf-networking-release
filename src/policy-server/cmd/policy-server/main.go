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

	"lib/nonmutualtls"
	"lib/poller"

	"policy-server/adapter"
	"policy-server/api"
	"policy-server/api/api_v0"
	"policy-server/cc_client"
	"policy-server/cleaner"
	"policy-server/cmd/common"
	"policy-server/config"
	"policy-server/handlers"
	"policy-server/store"
	"policy-server/uaa_client"

	"policy-server/store/migrations"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	middleware_adapter "code.cloudfoundry.org/cf-networking-helpers/middleware/adapter"
	"code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/dropsonde"
	"github.com/jmoiron/sqlx"
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

	logger := lager.NewLogger(fmt.Sprintf("%s.%s", logPrefix, jobPrefix))
	reconfigurableSink := common.InitLoggerSink(logger, conf.LogLevel)
	logger.RegisterSink(reconfigurableSink)

	var tlsConfig *tls.Config
	if conf.SkipSSLValidation {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: conf.SkipSSLValidation,
		}
	} else {
		tlsConfig, err = nonmutualtls.NewClientTLSConfig(conf.UAACA)
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

	logger.Info("getting migration db connection", lager.Data{})
	migrationConnectionResult := getMigrationDbConnection(*conf)
	logger.Info("migration db connection retrieved", lager.Data{})

	timeout := time.Duration(conf.Database.Timeout) * time.Second

	logger.Info("getting db connection", lager.Data{})
	connectionResult := getDbConnection(*conf)
	logger.Info("db connection retrieved", lager.Data{})

	dataStore, err := store.New(
		connectionResult.ConnectionPool,
		migrationConnectionResult.ConnectionPool,
		storeGroup,
		destination,
		policy,
		conf.TagLength,
		timeout,
		&migrations.Migrator{
			MigrateAdapter: &migrations.MigrateAdapter{},
		},
	)
	if err != nil {
		log.Fatalf("%s.%s: failed to construct datastore: %s", logPrefix, jobPrefix, err) // not tested
	}

	metricsSender := &metrics.MetricsSender{
		Logger: logger.Session("time-metric-emitter"),
	}

	wrappedStore := &store.MetricsWrapper{
		Store:         dataStore,
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

	policyMapperV0 := api_v0.NewMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal), &api_v0.Validator{})
	policyMapperV1 := api.NewMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal), &api.Validator{})

	createPolicyHandlerV1 := handlers.NewPoliciesCreate(wrappedStore, policyMapperV1,
		policyGuard, quotaGuard, errorResponse)
	createPolicyHandlerV0 := handlers.NewPoliciesCreate(wrappedStore, policyMapperV0,
		policyGuard, quotaGuard, errorResponse)

	deletePolicyHandlerV1 := handlers.NewPoliciesDelete(wrappedStore, policyMapperV1,
		policyGuard, errorResponse)
	deletePolicyHandlerV0 := handlers.NewPoliciesDelete(wrappedStore, policyMapperV0,
		policyGuard, errorResponse)

	policiesIndexHandlerV1 := handlers.NewPoliciesIndex(wrappedStore, policyMapperV1, policyFilter, errorResponse)
	policiesIndexHandlerV0 := handlers.NewPoliciesIndex(wrappedStore, policyMapperV0, policyFilter, errorResponse)

	policyCleaner := cleaner.NewPolicyCleaner(logger.Session("policy-cleaner"), wrappedStore, uaaClient,
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
		UUIDGenerator: &middleware_adapter.UUIDAdapter{},
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

	externalHandlers := rata.Handlers{
		"uptime": metricsWrap("Uptime", logWrap(uptimeHandler)),
		"health": metricsWrap("Health", logWrap(healthHandler)),

		"create_policies": metricsWrap("CreatePolicies",
			logWrap(versionWrap(authWriteWrap(createPolicyHandlerV1), authWriteWrap(createPolicyHandlerV0)))),

		"delete_policies": metricsWrap("DeletePolicies",
			logWrap(versionWrap(authWriteWrap(deletePolicyHandlerV1), authWriteWrap(deletePolicyHandlerV0)))),

		"policies_index": metricsWrap("PoliciesIndex",
			logWrap(versionWrap(authWriteWrap(policiesIndexHandlerV1), authWriteWrap(policiesIndexHandlerV0)))),

		"cleanup": metricsWrap("Cleanup",
			logWrap(versionWrap(authAdminWrap(policiesCleanupHandler), authAdminWrap(policiesCleanupHandler)))),

		"tags_index": metricsWrap("TagsIndex",
			logWrap(versionWrap(authAdminWrap(tagsIndexHandler), authAdminWrap(tagsIndexHandler)))),

		"whoami": metricsWrap("WhoAmI",
			logWrap(versionWrap(authAdminWrap(whoamiHandler), authAdminWrap(whoamiHandler)))),
	}

	externalRoutes := rata.Routes{
		{Name: "uptime", Method: "GET", Path: "/"},
		{Name: "uptime", Method: "GET", Path: "/networking"},
		{Name: "health", Method: "GET", Path: "/health"},
		{Name: "whoami", Method: "GET", Path: "/networking/:version/external/whoami"},
		{Name: "create_policies", Method: "POST", Path: "/networking/:version/external/policies"},
		{Name: "delete_policies", Method: "POST", Path: "/networking/:version/external/policies/delete"},
		{Name: "policies_index", Method: "GET", Path: "/networking/:version/external/policies"},
		{Name: "cleanup", Method: "POST", Path: "/networking/:version/external/policies/cleanup"},
		{Name: "tags_index", Method: "GET", Path: "/networking/:version/external/tags"},
	}

	err = dropsonde.Initialize(conf.MetronAddress, dropsondeOrigin)
	if err != nil {
		log.Fatalf("%s.%s: initializing dropsonde: %s", logPrefix, jobPrefix, err)
	}

	metricsEmitter := common.InitMetricsEmitter(logger, wrappedStore)
	externalServer := common.InitServer(logger, nil, conf.ListenHost, conf.ListenPort, externalHandlers, externalRoutes)
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
	if connectionResult.ConnectionPool != nil {
		connectionResult.ConnectionPool.Close()
	}
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}

	logger.Info("exited")
}
func getMigrationDbConnection(c config.Config) dbConnection {
	return getDbConnection(c)
}

type dbConnection struct {
	ConnectionPool *sqlx.DB
	Err            error
}

func getDbConnection(conf config.Config) dbConnection {
	retriableConnector := db.RetriableConnector{
		Connector:     db.GetConnectionPool,
		Sleeper:       db.SleeperFunc(time.Sleep),
		RetryInterval: 3 * time.Second,
		MaxRetries:    10,
	}

	channel := make(chan dbConnection)
	go func() {
		connection, err := retriableConnector.GetConnectionPool(conf.Database)
		channel <- dbConnection{connection, err}
	}()
	var connectionResult dbConnection
	select {
	case connectionResult = <-channel:
	case <-time.After(time.Duration(conf.Database.Timeout) * time.Second):
		log.Fatalf("%s.policy-server: db connection timeout", logPrefix)
	}
	if connectionResult.Err != nil {
		log.Fatalf("%s.policy-server: db connect: %s", logPrefix, connectionResult.Err) // not tested
	}
	return connectionResult
}

func initPoller(logger lager.Logger, conf *config.Config, policyCleaner *cleaner.PolicyCleaner) ifrit.Runner {
	pollInterval := time.Duration(conf.CleanupInterval) * time.Second

	return &poller.Poller{
		Logger:          logger.Session("policy-cleaner-poller"),
		PollInterval:    pollInterval,
		SingleCycleFunc: policyCleaner.DeleteStalePoliciesWrapper,
	}
}
