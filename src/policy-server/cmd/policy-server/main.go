package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"lib/nonmutualtls"
	"lib/poller"

	"policy-server/api"
	"policy-server/api/api_v0"
	"policy-server/api/api_v0_internal"
	"policy-server/cc_client"
	"policy-server/cleaner"
	"policy-server/config"
	"policy-server/handlers"
	"policy-server/server_metrics"
	"policy-server/store"
	"policy-server/uaa_client"

	"policy-server/store/migrations"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	"code.cloudfoundry.org/cf-networking-helpers/mutualtls"
	"code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/dropsonde"
	"github.com/jmoiron/sqlx"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
	"github.com/tedsuo/rata"
)

const (
	dropsondeOrigin = "policy-server"
	emitInterval    = 30 * time.Second
)

var (
	logPrefix = "cfnetworking"
)

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	conf, err := config.New(*configFilePath)
	if err != nil {
		log.Fatalf("%s.policy-server: could not read config file: %s", logPrefix, err)
	}

	if conf.LogPrefix != "" {
		logPrefix = conf.LogPrefix
	}

	logger := lager.NewLogger(fmt.Sprintf("%s.policy-server", logPrefix))
	reconfigurableSink := initLoggerSink(logger, conf.LogLevel)
	logger.RegisterSink(reconfigurableSink)

	var tlsConfig *tls.Config
	if conf.SkipSSLValidation {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: conf.SkipSSLValidation,
		}
	} else {
		tlsConfig, err = nonmutualtls.NewClientTLSConfig(conf.UAACA)
		if err != nil {
			log.Fatalf("%s.policy-server error creating tls config: %s", logPrefix, err) // not tested
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

	retriableConnector := db.RetriableConnector{
		Connector:     db.GetConnectionPool,
		Sleeper:       db.SleeperFunc(time.Sleep),
		RetryInterval: 3 * time.Second,
		MaxRetries:    10,
	}

	type dbConnection struct {
		ConnectionPool *sqlx.DB
		Err            error
	}
	channel := make(chan dbConnection)
	go func() {
		connection, err := retriableConnector.GetConnectionPool(conf.Database)
		channel <- dbConnection{connection, err}
	}()
	var connectionResult dbConnection
	select {
	case connectionResult = <-channel:
	case <-time.After(5 * time.Second):
		log.Fatalf("%s.policy-server: db connection timeout", logPrefix)
	}
	if connectionResult.Err != nil {
		log.Fatalf("%s.policy-server: db connect: %s", logPrefix, connectionResult.Err) // not tested
	}

	timeout := time.Duration(conf.Database.Timeout) * time.Second
	timeout = timeout - time.Duration(500)*time.Millisecond

	dataStore, err := store.New(
		connectionResult.ConnectionPool,
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
		log.Fatalf("%s.policy-server: failed to construct datastore: %s", logPrefix, err) // not tested
	}

	metricsSender := &metrics.MetricsSender{
		Logger: logger.Session("time-metric-emitter"),
	}

	wrappedStore := &store.MetricsWrapper{
		Store:         dataStore,
		MetricsSender: metricsSender,
	}

	errorResponse := &httperror.ErrorResponse{
		Logger:        logger,
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
	policyMapperV0Internal := api_v0_internal.NewMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal))
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

	internalPoliciesHandlerV0 := handlers.NewPoliciesIndexInternal(logger, wrappedStore,
		policyMapperV0Internal, errorResponse)
	internalPoliciesHandlerV1 := handlers.NewPoliciesIndexInternal(logger, wrappedStore,
		policyMapperV1, errorResponse)

	healthHandler := handlers.NewHealth(wrappedStore, errorResponse)

	checkVersionWrapper := &handlers.CheckVersionWrapper{
		ErrorResponse: errorResponse,
	}

	metricsWrap := func(name string, handler http.Handler) http.Handler {
		metricsWrapper := middleware.MetricWrapper{
			Name:          name,
			MetricsSender: metricsSender,
		}
		return metricsWrapper.Wrap(handler)
	}

	logWrap := func(handler http.HandlerFunc) http.HandlerFunc {
		return middleware.LogWrap(logger, handler)
	}

	versionWrap := func(v1Handler, v0Handler http.HandlerFunc) http.HandlerFunc {
		return checkVersionWrapper.CheckVersion(map[string]http.HandlerFunc{
			"v1": v1Handler,
			"v0": v0Handler,
		})
	}

	authAdminWrap := func(handler http.HandlerFunc) http.HandlerFunc {
		networkAdminAuthenticator := handlers.Authenticator{
			Client:        uaaClient,
			Scopes:        []string{"network.admin"},
			ErrorResponse: errorResponse,
			ScopeChecking: true,
		}
		return networkAdminAuthenticator.Wrap(handler)
	}

	authWriteWrap := func(handler http.HandlerFunc) http.HandlerFunc {
		networkWriteAuthenticator := handlers.Authenticator{
			Client:        uaaClient,
			Scopes:        []string{"network.admin", "network.write"},
			ErrorResponse: errorResponse,
			ScopeChecking: !conf.EnableSpaceDeveloperSelfService,
		}
		return networkWriteAuthenticator.Wrap(handler)
	}

	externalHandlers := rata.Handlers{
		"uptime": metricsWrap("Uptime", logWrap(uptimeHandler.ServeHTTP)),
		"health": metricsWrap("Health", logWrap(healthHandler.ServeHTTP)),

		"create_policies": metricsWrap("CreatePolicies",
			logWrap(versionWrap(authWriteWrap(createPolicyHandlerV1.ServeHTTP), authWriteWrap(createPolicyHandlerV0.ServeHTTP)))),

		"delete_policies": metricsWrap("DeletePolicies",
			logWrap(versionWrap(authWriteWrap(deletePolicyHandlerV1.ServeHTTP), authWriteWrap(deletePolicyHandlerV0.ServeHTTP)))),

		"policies_index": metricsWrap("PoliciesIndex",
			logWrap(versionWrap(authWriteWrap(policiesIndexHandlerV1.ServeHTTP), authWriteWrap(policiesIndexHandlerV0.ServeHTTP)))),

		"cleanup": metricsWrap("Cleanup",
			logWrap(versionWrap(authAdminWrap(policiesCleanupHandler.ServeHTTP), authAdminWrap(policiesCleanupHandler.ServeHTTP)))),

		"tags_index": metricsWrap("TagsIndex",
			logWrap(versionWrap(authAdminWrap(tagsIndexHandler.ServeHTTP), authAdminWrap(tagsIndexHandler.ServeHTTP)))),

		"whoami": metricsWrap("WhoAmI",
			logWrap(versionWrap(authAdminWrap(whoamiHandler.ServeHTTP), authAdminWrap(whoamiHandler.ServeHTTP)))),
	}

	err = dropsonde.Initialize(conf.MetronAddress, dropsondeOrigin)
	if err != nil {
		log.Fatalf("%s.policy-server: initializing dropsonde: %s", logPrefix, err)
	}

	metricsEmitter := initMetricsEmitter(logger, wrappedStore)
	externalServer := initExternalServer(conf, externalHandlers)
	internalServer := initInternalServer(conf, metricsWrap("InternalPolicies", logWrap(
		versionWrap(internalPoliciesHandlerV1.ServeHTTP, internalPoliciesHandlerV0.ServeHTTP),
	)))
	poller := initPoller(logger, conf, policyCleaner)
	debugServer := debugserver.Runner(fmt.Sprintf("%s:%d", conf.DebugServerHost, conf.DebugServerPort), reconfigurableSink)

	members := grouper.Members{
		{"metrics_emitter", metricsEmitter},
		{"http_server", externalServer},
		{"internal_http_server", internalServer},
		{"policy-cleaner-poller", poller},
		{"debug-server", debugServer},
	}

	logger.Info("starting external server", lager.Data{"listen-address": conf.ListenHost, "port": conf.ListenPort})
	logger.Info("starting internal server", lager.Data{"listen-address": conf.ListenHost, "port": conf.InternalListenPort})

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

const (
	DEBUG = "debug"
	INFO  = "info"
	ERROR = "error"
	FATAL = "fatal"
)

func initLoggerSink(logger lager.Logger, level string) *lager.ReconfigurableSink {
	var logLevel lager.LogLevel
	switch strings.ToLower(level) {
	case DEBUG:
		logLevel = lager.DEBUG
	case INFO:
		logLevel = lager.INFO
	case ERROR:
		logLevel = lager.ERROR
	case FATAL:
		logLevel = lager.FATAL
	default:
		logLevel = lager.INFO
	}
	w := lager.NewWriterSink(os.Stdout, lager.DEBUG)
	return lager.NewReconfigurableSink(w, logLevel)
}

func initMetricsEmitter(logger lager.Logger, wrappedStore *store.MetricsWrapper) *metrics.MetricsEmitter {
	totalPoliciesSource := server_metrics.NewTotalPoliciesSource(wrappedStore)
	uptimeSource := metrics.NewUptimeSource()
	return metrics.NewMetricsEmitter(logger, emitInterval, uptimeSource, totalPoliciesSource)
}

func initPoller(logger lager.Logger, conf *config.Config, policyCleaner *cleaner.PolicyCleaner) ifrit.Runner {
	pollInterval := time.Duration(conf.CleanupInterval) * time.Second

	return &poller.Poller{
		Logger:          logger.Session("policy-cleaner-poller"),
		PollInterval:    pollInterval,
		SingleCycleFunc: policyCleaner.DeleteStalePoliciesWrapper,
	}
}

func initInternalServer(conf *config.Config, internalPoliciesHandler http.Handler) ifrit.Runner {
	routes := rata.Routes{
		{Name: "internal_policies", Method: "GET", Path: "/networking/:version/internal/policies"},
	}
	handlers := rata.Handlers{
		"internal_policies": internalPoliciesHandler,
	}

	router, err := rata.NewRouter(routes, handlers)
	if err != nil {
		log.Fatalf("%s.policy-server: unable to create rata Router: %s", logPrefix, err) // not tested
	}

	addr := fmt.Sprintf("%s:%d", conf.ListenHost, conf.InternalListenPort)

	tlsConfig, err := mutualtls.NewServerTLSConfig(conf.ServerCertFile, conf.ServerKeyFile, conf.CACertFile)
	if err != nil {
		log.Fatalf("%s.policy-server: mutual tls config: %s", logPrefix, err) // not tested
	}

	return http_server.NewTLSServer(addr, router, tlsConfig)
}

func initExternalServer(conf *config.Config, externalHandlers rata.Handlers) ifrit.Runner {
	routes := rata.Routes{
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

	externalRouter, err := rata.NewRouter(routes, externalHandlers)
	if err != nil {
		log.Fatalf("%s.policy-server: unable to create rata Router: %s", logPrefix, err) // not tested
	}

	addr := fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort)
	return http_server.New(addr, externalRouter)
}
