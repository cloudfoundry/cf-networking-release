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

	"lib/db"
	"lib/json_client"
	"lib/marshal"
	"lib/metrics"
	"lib/mutualtls"
	"lib/poller"

	"policy-server/cc_client"
	"policy-server/cleaner"
	"policy-server/config"
	"policy-server/handlers"
	"policy-server/server_metrics"
	"policy-server/store"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/dropsonde"
	"github.com/jmoiron/sqlx"
	"github.com/pivotal-cf-experimental/warrant"
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

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: conf.SkipSSLValidation,
			},
		},
	}

	warrantClient := warrant.New(warrant.Config{
		Host:          conf.UAAURL,
		SkipVerifySSL: conf.SkipSSLValidation,
	})

	uaaClient := &uaa_client.Client{
		BaseURL:       conf.UAAURL,
		Name:          conf.UAAClient,
		Secret:        conf.UAAClientSecret,
		HTTPClient:    httpClient,
		WarrantClient: warrantClient.Clients,
		Logger:        logger,
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

	unmarshaler := marshal.UnmarshalFunc(json.Unmarshal)

	authenticator := handlers.Authenticator{
		Client: uaaClient,
		Logger: logger,
		Scopes: []string{"network.admin"},
	}

	networkWriteAuthenticator := handlers.Authenticator{
		Client: uaaClient,
		Logger: logger,
		Scopes: []string{"network.admin", "network.write"},
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
		Logger:      logger.Session("policies-create"),
		Store:       dataStore,
		Unmarshaler: unmarshaler,
		Validator:   validator,
		PolicyGuard: policyGuard,
	}

	deletePolicyHandler := &handlers.PoliciesDelete{
		Logger:      logger.Session("policies-create"),
		Store:       dataStore,
		Unmarshaler: unmarshaler,
		Validator:   validator,
		PolicyGuard: policyGuard,
	}

	policiesIndexHandler := &handlers.PoliciesIndex{
		Logger:       logger.Session("policies-index"),
		Store:        dataStore,
		Marshaler:    marshal.MarshalFunc(json.Marshal),
		PolicyFilter: policyFilter,
	}

	policyCleaner := &cleaner.PolicyCleaner{
		Logger:    logger.Session("policy-cleaner"),
		Store:     dataStore,
		UAAClient: uaaClient,
		CCClient:  ccClient,
	}

	policiesCleanupHandler := &handlers.PoliciesCleanup{
		Logger:        logger.Session("policies-cleanup"),
		Marshaler:     marshal.MarshalFunc(json.Marshal),
		PolicyCleaner: policyCleaner,
	}

	tagsIndexHandler := &handlers.TagsIndex{
		Logger:    logger.Session("tags-index"),
		Store:     dataStore,
		Marshaler: marshal.MarshalFunc(json.Marshal),
	}

	timeMetricsEmitter := &server_metrics.TimeMetrics{
		Logger: logger.Session("time-metric-emitter"),
	}
	internalPoliciesHandler := &handlers.PoliciesIndexInternal{
		Logger:         logger.Session("policies-index-internal"),
		Store:          dataStore,
		Marshaler:      marshal.MarshalFunc(json.Marshal),
		MetricsEmitter: timeMetricsEmitter,
	}

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

	handlers := rata.Handlers{
		"uptime":          uptimeHandler,
		"create_policies": networkWriteAuthenticator.Wrap(createPolicyHandler),
		"delete_policies": networkWriteAuthenticator.Wrap(deletePolicyHandler),
		"policies_index":  networkWriteAuthenticator.Wrap(policiesIndexHandler),
		"cleanup":         authenticator.Wrap(policiesCleanupHandler),
		"tags_index":      authenticator.Wrap(tagsIndexHandler),
		"whoami":          authenticator.Wrap(whoamiHandler),
	}
	router, err := rata.NewRouter(routes, handlers)
	if err != nil {
		log.Fatalf("unable to create rata Router: %s", err) // not tested
	}

	addr := fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort)
	server := http_server.New(addr, router)

	internalRoutes := rata.Routes{
		{Name: "internal_policies", Method: "GET", Path: "/networking/v0/internal/policies"},
	}

	internalHandlers := rata.Handlers{
		"internal_policies": internalPoliciesHandler,
	}
	internalRouter, err := rata.NewRouter(internalRoutes, internalHandlers)
	if err != nil {
		log.Fatalf("unable to create rata Router: %s", err)
	}
	internalAddr := fmt.Sprintf("%s:%d", conf.ListenHost, conf.InternalListenPort)

	tlsConfig, err := mutualtls.NewServerTLSConfig(conf.ServerCertFile, conf.ServerKeyFile, conf.CACertFile)
	if err != nil {
		log.Fatalf("mutual tls config: %s", err)
	}
	internalServer := http_server.NewTLSServer(internalAddr, internalRouter, tlsConfig)

	err = dropsonde.Initialize(conf.MetronAddress, dropsondeOrigin)
	if err != nil {
		log.Fatalf("initializing dropsonde: %s", err)
	}

	totalPoliciesSource := server_metrics.NewTotalPoliciesSource(dataStore)
	uptimeSource := metrics.NewUptimeSource()
	metricsEmitter := metrics.NewMetricsEmitter(logger, emitInterval, uptimeSource, totalPoliciesSource)

	pollInterval := time.Duration(conf.CleanupInterval) * time.Second

	poller := &poller.Poller{
		Logger:          logger.Session("policy-cleaner-poller"),
		PollInterval:    pollInterval,
		SingleCycleFunc: policyCleaner.DeleteStalePoliciesWrapper,
	}

	debugServerAddress := fmt.Sprintf("%s:%d", conf.DebugServerHost, conf.DebugServerPort)
	members := grouper.Members{
		{"metrics_emitter", metricsEmitter},
		{"http_server", server},
		{"internal_http_server", internalServer},
		{"policy-cleaner-poller", poller},
		{"debug-server", debugserver.Runner(debugServerAddress, reconfigurableSink)},
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
