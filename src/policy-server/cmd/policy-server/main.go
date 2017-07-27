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
	"policy-server/api/api_0_0_0"
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

	policyGuard := &handlers.PolicyGuard{
		UAAClient: uaaClient,
		CCClient:  ccClient,
	}

	quotaGuard := &handlers.QuotaGuard{
		Store:       wrappedStore,
		MaxPolicies: conf.MaxPolicies,
	}

	policyFilter := &handlers.PolicyFilter{
		UAAClient: uaaClient,
		CCClient:  ccClient,
	}

	policyMapperV0 := api_0_0_0.NewMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal))
	policyMapperV1 := api.NewMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal))

	validator := &handlers.Validator{}

	createPolicyHandlerV1 := &handlers.PoliciesCreate{
		Store:         wrappedStore,
		Mapper:        policyMapperV1,
		Validator:     validator,
		PolicyGuard:   policyGuard,
		QuotaGuard:    quotaGuard,
		ErrorResponse: errorResponse,
	}

	createPolicyHandlerV0 := &handlers.PoliciesCreate{
		Store:         wrappedStore,
		Mapper:        policyMapperV0,
		Validator:     validator,
		PolicyGuard:   policyGuard,
		QuotaGuard:    quotaGuard,
		ErrorResponse: errorResponse,
	}

	deletePolicyHandlerV1 := &handlers.PoliciesDelete{
		Store:         wrappedStore,
		Mapper:        policyMapperV1,
		Validator:     validator,
		PolicyGuard:   policyGuard,
		ErrorResponse: errorResponse,
	}

	deletePolicyHandlerV0 := &handlers.PoliciesDelete{
		Store:         wrappedStore,
		Mapper:        policyMapperV0,
		Validator:     validator,
		PolicyGuard:   policyGuard,
		ErrorResponse: errorResponse,
	}

	policiesIndexHandlerV1 := &handlers.PoliciesIndex{
		Store:         wrappedStore,
		Mapper:        policyMapperV1,
		PolicyFilter:  policyFilter,
		ErrorResponse: errorResponse,
	}

	policiesIndexHandlerV0 := &handlers.PoliciesIndex{
		Store:         wrappedStore,
		Mapper:        policyMapperV0,
		PolicyFilter:  policyFilter,
		ErrorResponse: errorResponse,
	}

	policyCleaner := &cleaner.PolicyCleaner{
		Logger:         logger.Session("policy-cleaner"),
		Store:          wrappedStore,
		UAAClient:      uaaClient,
		CCClient:       ccClient,
		RequestTimeout: time.Duration(5) * time.Second,
	}

	policiesCleanupHandler := &handlers.PoliciesCleanup{
		Mapper:        policyMapperV1,
		PolicyCleaner: policyCleaner,
		ErrorResponse: errorResponse,
	}

	tagsIndexHandler := &handlers.TagsIndex{
		Store:         wrappedStore,
		Marshaler:     marshal.MarshalFunc(json.Marshal),
		ErrorResponse: errorResponse,
	}

	internalPoliciesHandler := &handlers.PoliciesIndexInternal{
		Logger:        logger.Session("policies-index-internal"),
		Store:         wrappedStore,
		Mapper:        policyMapperV1,
		ErrorResponse: errorResponse,
	}

	healthHandler := &handlers.Health{
		Store:         wrappedStore,
		ErrorResponse: errorResponse,
	}

	metricsWrap := func(name string, handler http.Handler) http.Handler {
		metricsWrapper := middleware.MetricWrapper{
			Name:          name,
			MetricsSender: metricsSender,
		}
		return metricsWrapper.Wrap(handler)
	}

	type loggableHandler interface {
		ServeHTTP(logger lager.Logger, w http.ResponseWriter, r *http.Request)
	}
	logWrap := func(handler loggableHandler) http.Handler {
		return middleware.LogWrap(logger, handler.ServeHTTP)
	}

	authenticator := handlers.Authenticator{
		Client:        uaaClient,
		Scopes:        []string{"network.admin"},
		ErrorResponse: errorResponse,
		ScopeChecking: true,
	}

	networkWriteAuthenticator := handlers.Authenticator{
		Client:        uaaClient,
		Scopes:        []string{"network.admin", "network.write"},
		ErrorResponse: errorResponse,
		ScopeChecking: !conf.EnableSpaceDeveloperSelfService,
	}
	authAdmin := func(handler handlers.AuthenticatedHandler) middleware.LoggableHandlerFunc {
		return authenticator.Wrap(handler)
	}
	authWrite := func(handler handlers.AuthenticatedHandler) middleware.LoggableHandlerFunc {
		return networkWriteAuthenticator.Wrap(handler)
	}

	checkVersionWrapper := &handlers.CheckVersionWrapper{
		ErrorResponse: errorResponse,
	}

	externalHandlers := rata.Handlers{
		"uptime": metricsWrap("Uptime", logWrap(uptimeHandler)),
		"health": metricsWrap("Health", logWrap(healthHandler)),
		"create_policies": metricsWrap("CreatePolicies", middleware.LogWrap(logger,
			checkVersionWrapper.CheckVersion(map[string]middleware.LoggableHandlerFunc{
				"1.0.0": authWrite(createPolicyHandlerV1),
				"0.0.0": authWrite(createPolicyHandlerV0),
			}),
		)),
		"delete_policies": metricsWrap("DeletePolicies", middleware.LogWrap(logger,
			checkVersionWrapper.CheckVersion(map[string]middleware.LoggableHandlerFunc{
				"1.0.0": authWrite(deletePolicyHandlerV1),
				"0.0.0": authWrite(deletePolicyHandlerV0),
			}),
		)),
		"policies_index": metricsWrap("PoliciesIndex", middleware.LogWrap(logger,
			checkVersionWrapper.CheckVersion(map[string]middleware.LoggableHandlerFunc{
				"1.0.0": authWrite(policiesIndexHandlerV1),
				"0.0.0": authWrite(policiesIndexHandlerV0),
			}),
		)),
		"cleanup":    metricsWrap("Cleanup", middleware.LogWrap(logger, authAdmin(policiesCleanupHandler))),
		"tags_index": metricsWrap("TagsIndex", middleware.LogWrap(logger, authAdmin(tagsIndexHandler))),
		"whoami":     metricsWrap("WhoAmI", middleware.LogWrap(logger, authAdmin(whoamiHandler))),
	}

	err = dropsonde.Initialize(conf.MetronAddress, dropsondeOrigin)
	if err != nil {
		log.Fatalf("%s.policy-server: initializing dropsonde: %s", logPrefix, err)
	}

	metricsEmitter := initMetricsEmitter(logger, wrappedStore)
	externalServer := initExternalServer(conf, externalHandlers)
	internalServer := initInternalServer(conf, metricsWrap("InternalPolicies", logWrap(internalPoliciesHandler)))
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
		{Name: "internal_policies", Method: "GET", Path: "/networking/v0/internal/policies"},
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
		{Name: "whoami", Method: "GET", Path: "/networking/v0/external/whoami"},
		{Name: "create_policies", Method: "POST", Path: "/networking/v0/external/policies"},
		{Name: "delete_policies", Method: "POST", Path: "/networking/v0/external/policies/delete"},
		{Name: "policies_index", Method: "GET", Path: "/networking/v0/external/policies"},
		{Name: "cleanup", Method: "POST", Path: "/networking/v0/external/policies/cleanup"},
		{Name: "tags_index", Method: "GET", Path: "/networking/v0/external/tags"},
	}

	externalRouter, err := rata.NewRouter(routes, externalHandlers)
	if err != nil {
		log.Fatalf("%s.policy-server: unable to create rata Router: %s", logPrefix, err) // not tested
	}

	addr := fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort)
	return http_server.New(addr, externalRouter)
}
