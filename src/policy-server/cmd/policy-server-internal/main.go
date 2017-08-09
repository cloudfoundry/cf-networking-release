package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"policy-server/api"
	"policy-server/api/api_v0_internal"
	"policy-server/cmd/common"
	"policy-server/config"
	"policy-server/handlers"
	"policy-server/store"

	"policy-server/store/migrations"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/httperror"
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
	"github.com/tedsuo/ifrit/sigmon"
	"github.com/tedsuo/rata"
)

const (
	jobPrefix = "policy-server-internal"
)

var (
	logPrefix = "cfnetworking"
)

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	conf, err := config.NewInternal(*configFilePath)
	if err != nil {
		log.Fatalf("%s.%s: could not read config file: %s", logPrefix, jobPrefix, err)
	}

	if conf.LogPrefix != "" {
		logPrefix = conf.LogPrefix
	}

	logger := lager.NewLogger(fmt.Sprintf("%s.%s", logPrefix, jobPrefix))
	reconfigurableSink := common.InitLoggerSink(logger, conf.LogLevel)
	logger.RegisterSink(reconfigurableSink)

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
		log.Fatalf("%s.%s: db connection timeout", logPrefix)
	}
	if connectionResult.Err != nil {
		log.Fatalf("%s.%s: db connect: %s", logPrefix, jobPrefix, connectionResult.Err) // not tested
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
		Logger:        logger,
		MetricsSender: metricsSender,
	}

	policyMapperV0Internal := api_v0_internal.NewMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal))
	policyMapperV1 := api.NewMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal), &api.Validator{})

	internalPoliciesHandlerV0 := handlers.NewPoliciesIndexInternal(logger, wrappedStore,
		policyMapperV0Internal, errorResponse)
	internalPoliciesHandlerV1 := handlers.NewPoliciesIndexInternal(logger, wrappedStore,
		policyMapperV1, errorResponse)

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

	logWrap := func(handler http.Handler) http.Handler {
		return middleware.LogWrap(logger, handler)
	}

	versionWrap := func(v1Handler, v0Handler http.Handler) http.Handler {
		return checkVersionWrapper.CheckVersion(map[string]http.Handler{
			"v1": v1Handler,
			"v0": v0Handler,
		})
	}

	err = dropsonde.Initialize(conf.MetronAddress, jobPrefix)
	if err != nil {
		log.Fatalf("%s.%s: initializing dropsonde: %s", logPrefix, jobPrefix, err)
	}

	metricsEmitter := common.InitMetricsEmitter(logger, wrappedStore)

	internalRoutes := rata.Routes{
		{Name: "internal_policies", Method: "GET", Path: "/networking/:version/internal/policies"},
	}
	internalHandlers := rata.Handlers{
		"internal_policies": metricsWrap("InternalPolicies", logWrap(
			versionWrap(internalPoliciesHandlerV1, internalPoliciesHandlerV0),
		)),
	}

	tlsConfig, err := mutualtls.NewServerTLSConfig(conf.ServerCertFile, conf.ServerKeyFile, conf.CACertFile)
	if err != nil {
		log.Fatalf("%s.%s: mutual tls config: %s", logPrefix, jobPrefix, err) // not tested
	}

	internalServer := common.InitServer(logger, tlsConfig, conf.ListenHost, conf.InternalListenPort, internalHandlers, internalRoutes)
	debugServer := debugserver.Runner(fmt.Sprintf("%s:%d", conf.DebugServerHost, conf.DebugServerPort), reconfigurableSink)

	uptimeHandler := &handlers.UptimeHandler{
		StartTime: time.Now(),
	}
	healthHandler := handlers.NewHealth(wrappedStore, errorResponse)

	healthRoutes := rata.Routes{
		{Name: "uptime", Method: "GET", Path: "/"},
		{Name: "health", Method: "GET", Path: "/health"},
	}

	healthHandlers := rata.Handlers{
		"uptime": metricsWrap("Uptime", logWrap(uptimeHandler)),
		"health": metricsWrap("Health", logWrap(healthHandler)),
	}

	healthCheckServer := common.InitServer(logger, nil, conf.ListenHost,
		conf.HealthCheckPort, healthHandlers, healthRoutes)

	members := grouper.Members{
		{"metrics-emitter", metricsEmitter},
		{"internal-http-server", internalServer},
		{"debug-server", debugServer},
		{"health-check-server", healthCheckServer},
	}

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
