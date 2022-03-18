package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	middlewareAdapter "code.cloudfoundry.org/cf-networking-helpers/middleware/adapter"
	"code.cloudfoundry.org/cf-networking-helpers/mutualtls"
	"code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/lib/common"
	"code.cloudfoundry.org/policy-server/api"
	"code.cloudfoundry.org/policy-server/config"
	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/store"
	"github.com/cloudfoundry/dropsonde"
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

	loggerConfig := common.GetLagerConfig()
	if conf.LogLevel != "" {
		loggerConfig.LogLevel = conf.LogLevel
	}
	logger, reconfigurableSink := lagerflags.NewFromConfig(fmt.Sprintf("%s.%s", logPrefix, jobPrefix), loggerConfig)

	connectionPool, err := db.NewConnectionPool(
		conf.Database,
		conf.MaxOpenConnections,
		conf.MaxIdleConnections,
		time.Duration(conf.MaxConnectionsLifetimeSeconds)*time.Second,
		logPrefix,
		jobPrefix,
		logger,
	)
	if err != nil {
		log.Fatalf(err.Error())
	}

	dataStore := store.New(
		connectionPool,
		&store.GroupTable{},
		&store.DestinationTable{},
		&store.PolicyTable{},
		conf.TagLength,
	)

	securityGroupsStore := &store.SGStore{
		Conn: connectionPool,
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

	wrappedSecurityGroupsStore := &store.SecurityGroupsMetricsWrapper{
		Store:         securityGroupsStore,
		MetricsSender: metricsSender,
	}

	errorResponse := &httperror.ErrorResponse{
		MetricsSender: metricsSender,
	}

	policyMapperWriter := api.NewPolicyMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal), &api.PolicyValidator{})

	internalPoliciesHandlerV1 := handlers.NewPoliciesIndexInternal(logger, wrappedStore, policyMapperWriter, errorResponse)

	createTagsHandlerV1 := &handlers.TagsCreate{
		Store:         wrappedStore,
		ErrorResponse: errorResponse,
	}

	asgMapper := api.NewAsgMapper(marshal.MarshalFunc(json.Marshal))
	securityGroupsHandlerV1 := handlers.NewAsgsIndex(wrappedSecurityGroupsStore, asgMapper, errorResponse)

	hstsHeaderWrapper := handlers.HSTSHandler{}

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

	err = dropsonde.Initialize(conf.MetronAddress, jobPrefix)
	if err != nil {
		log.Fatalf("%s.%s: initializing dropsonde: %s", logPrefix, jobPrefix, err)
	}

	metricsEmitter := common.InitMetricsEmitter(logger, wrappedStore, connectionPool, connectionPool.Monitor)

	internalRoutes := rata.Routes{
		{Name: "create_tags", Method: "PUT", Path: "/networking/v1/internal/tags"},
		{Name: "internal_policies", Method: "GET", Path: "/networking/:version/internal/policies"},
		{Name: "internal_security_groups", Method: "GET", Path: "/networking/:version/internal/security_groups"},
	}

	internalHandlers := rata.Handlers{
		"create_tags":              metricsWrap("CreateTags", logWrap(createTagsHandlerV1)),
		"internal_policies":        metricsWrap("InternalPolicies", logWrap(internalPoliciesHandlerV1)),
		"internal_security_groups": metricsWrap("InternalSecurityGroups", logWrap(securityGroupsHandlerV1)),
	}

	for key, handler := range internalHandlers {
		wrappedHandler := hstsHeaderWrapper.Wrap(handler)
		internalHandlers[key] = wrappedHandler
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
		{Name: "metrics-emitter", Runner: metricsEmitter},
		{Name: "internal-http-server", Runner: internalServer},
		{Name: "debug-server", Runner: debugServer},
		{Name: "health-check-server", Runner: healthCheckServer},
	}

	logger.Info("starting internal server", lager.Data{"listen-address": conf.ListenHost, "port": conf.InternalListenPort})

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
