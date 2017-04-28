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

	"lib/metrics"
	"lib/nonmutualtls"
	"lib/poller"

	"policy-server/adapter"
	"policy-server/cc_client"
	"policy-server/cleaner"
	"policy-server/config"
	"policy-server/handlers"
	"policy-server/server_metrics"
	"policy-server/store"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/go-db-helpers/db"
	"code.cloudfoundry.org/go-db-helpers/httperror"
	"code.cloudfoundry.org/go-db-helpers/json_client"
	"code.cloudfoundry.org/go-db-helpers/marshal"
	"code.cloudfoundry.org/go-db-helpers/mutualtls"
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

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	conf, err := config.New(*configFilePath)
	if err != nil {
		log.Fatalf("could not read config file %s", err)
	}

	logger := lager.NewLogger("container-networking.policy-server")
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
			log.Fatalf("error creating tls config: %s", err)
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
		Logger:    logger.Session("external"),
		Marshaler: marshal.MarshalFunc(json.Marshal),
	}
	uptimeHandler := &handlers.UptimeHandler{
		StartTime: time.Now(),
	}

	storeGroup := &store.Group{}
	destination := &store.Destination{}
	policy := &store.Policy{}

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
		log.Fatal("db connection timeout")
	}
	if connectionResult.Err != nil {
		log.Fatalf("db connect: %s", connectionResult.Err)
	}

	dataStore, err := store.New(
		connectionResult.ConnectionPool,
		storeGroup,
		destination,
		policy,
		conf.TagLength,
	)
	if err != nil {
		log.Fatalf("failed to construct datastore: %s", err)
	}

	metricsSender := &metrics.MetricsSender{
		Logger: logger.Session("time-metric-emitter"),
	}

	wrappedStore := &store.MetricsWrapper{
		Store:         dataStore,
		MetricsSender: metricsSender,
	}

	unmarshaler := marshal.UnmarshalFunc(json.Unmarshal)

	errorResponse := &httperror.ErrorResponse{
		Logger:        logger,
		MetricsSender: metricsSender,
	}

	authenticator := handlers.Authenticator{
		Client:        uaaClient,
		Logger:        logger,
		Scopes:        []string{"network.admin"},
		ErrorResponse: errorResponse,
	}

	networkWriteAuthenticator := handlers.Authenticator{
		Client:        uaaClient,
		Logger:        logger,
		Scopes:        []string{"network.admin", "network.write"},
		ErrorResponse: errorResponse,
	}

	ccClient := &cc_client.Client{
		JSONClient: json_client.New(logger.Session("cc-json-client"), httpClient, conf.CCURL),
		Logger:     logger,
	}

	policyGuard := &handlers.PolicyGuard{
		UAAClient: uaaClient,
		CCClient:  ccClient,
	}

	policyFilter := &handlers.PolicyFilter{
		UAAClient: uaaClient,
		CCClient:  ccClient,
	}

	validator := &handlers.Validator{}

	createPolicyHandler := &handlers.PoliciesCreate{
		Logger:        logger.Session("policies-create"),
		Store:         wrappedStore,
		Unmarshaler:   unmarshaler,
		Validator:     validator,
		PolicyGuard:   policyGuard,
		ErrorResponse: errorResponse,
	}

	deletePolicyHandler := &handlers.PoliciesDelete{
		Logger:        logger.Session("policies-create"),
		Store:         wrappedStore,
		Unmarshaler:   unmarshaler,
		Validator:     validator,
		PolicyGuard:   policyGuard,
		ErrorResponse: errorResponse,
	}

	policiesIndexHandler := &handlers.PoliciesIndex{
		Logger:        logger.Session("policies-index"),
		Store:         wrappedStore,
		Marshaler:     marshal.MarshalFunc(json.Marshal),
		PolicyFilter:  policyFilter,
		ErrorResponse: errorResponse,
	}

	policyCleaner := &cleaner.PolicyCleaner{
		Logger:         logger.Session("policy-cleaner"),
		Store:          wrappedStore,
		UAAClient:      uaaClient,
		CCClient:       ccClient,
		RequestTimeout: time.Duration(5) * time.Second,
		ContextAdapter: &adapter.ContextAdapter{},
	}

	policiesCleanupHandler := &handlers.PoliciesCleanup{
		Logger:        logger.Session("policies-cleanup"),
		Marshaler:     marshal.MarshalFunc(json.Marshal),
		PolicyCleaner: policyCleaner,
		ErrorResponse: errorResponse,
	}

	tagsIndexHandler := &handlers.TagsIndex{
		Logger:        logger.Session("tags-index"),
		Store:         wrappedStore,
		Marshaler:     marshal.MarshalFunc(json.Marshal),
		ErrorResponse: errorResponse,
	}

	internalPoliciesHandler := &handlers.PoliciesIndexInternal{
		Logger:        logger.Session("policies-index-internal"),
		Store:         wrappedStore,
		Marshaler:     marshal.MarshalFunc(json.Marshal),
		ErrorResponse: errorResponse,
	}

	metricsWrap := func(name string, handle http.Handler) http.Handler {
		metricsWrapper := handlers.MetricWrapper{
			Name:          name,
			MetricsSender: metricsSender,
		}
		return metricsWrapper.Wrap(handle)
	}

	contextWrapper := handlers.ContextWrapper{
		Duration:       time.Duration(conf.RequestTimeout) * time.Second,
		ContextAdapter: &adapter.ContextAdapter{},
	}

	externalHandlers := rata.Handlers{
		"uptime":          metricsWrap("Uptime", uptimeHandler),
		"create_policies": contextWrapper.Wrap(metricsWrap("CreatePolicies", networkWriteAuthenticator.Wrap(createPolicyHandler))),
		"delete_policies": contextWrapper.Wrap(metricsWrap("DeletePolicies", networkWriteAuthenticator.Wrap(deletePolicyHandler))),
		"policies_index":  metricsWrap("PoliciesIndex", networkWriteAuthenticator.Wrap(policiesIndexHandler)),
		"cleanup":         metricsWrap("Cleanup", authenticator.Wrap(policiesCleanupHandler)),
		"tags_index":      metricsWrap("TagsIndex", authenticator.Wrap(tagsIndexHandler)),
		"whoami":          metricsWrap("WhoAmI", authenticator.Wrap(whoamiHandler)),
	}

	err = dropsonde.Initialize(conf.MetronAddress, dropsondeOrigin)
	if err != nil {
		log.Fatalf("initializing dropsonde: %s", err)
	}

	metricsEmitter := initMetricsEmitter(logger, wrappedStore)
	externalServer := initExternalServer(conf, externalHandlers)
	internalServer := initInternalServer(conf, metricsWrap("InternalPolicies", internalPoliciesHandler))
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
		log.Fatalf("unable to create rata Router: %s", err)
	}

	addr := fmt.Sprintf("%s:%d", conf.ListenHost, conf.InternalListenPort)

	tlsConfig, err := mutualtls.NewServerTLSConfig(conf.ServerCertFile, conf.ServerKeyFile, conf.CACertFile)
	if err != nil {
		log.Fatalf("mutual tls config: %s", err)
	}

	return http_server.NewTLSServer(addr, router, tlsConfig)
}

func initExternalServer(conf *config.Config, externalHandlers rata.Handlers) ifrit.Runner {
	routes := rata.Routes{
		{Name: "uptime", Method: "GET", Path: "/"},
		{Name: "uptime", Method: "GET", Path: "/networking"},
		{Name: "whoami", Method: "GET", Path: "/networking/v0/external/whoami"},
		{Name: "create_policies", Method: "POST", Path: "/networking/v0/external/policies"},
		{Name: "delete_policies", Method: "POST", Path: "/networking/v0/external/policies/delete"},
		{Name: "policies_index", Method: "GET", Path: "/networking/v0/external/policies"},
		{Name: "cleanup", Method: "POST", Path: "/networking/v0/external/policies/cleanup"},
		{Name: "tags_index", Method: "GET", Path: "/networking/v0/external/tags"},
	}

	externalRouter, err := rata.NewRouter(routes, externalHandlers)
	if err != nil {
		log.Fatalf("unable to create rata Router: %s", err) // not tested
	}

	addr := fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort)
	return http_server.New(addr, externalRouter)
}
