package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"lib/common"
	"log"
	"net/http"
	"os"
	"time"

	"policy-server/adapter"
	"policy-server/api"
	"policy-server/api/api_v0_internal"
	"policy-server/config"
	"policy-server/handlers"
	"policy-server/store"

	"policy-server/db"

	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	middlewareAdapter "code.cloudfoundry.org/cf-networking-helpers/middleware/adapter"
	"code.cloudfoundry.org/cf-networking-helpers/mutualtls"
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

	logger, reconfigurableSink := lagerflags.NewFromConfig(fmt.Sprintf("%s.%s", logPrefix, jobPrefix), common.GetLagerConfig())

	connectionPool := db.NewConnectionPool(
		conf.Database,
		conf.MaxOpenConnections,
		conf.MaxIdleConnections,
		logPrefix,
		jobPrefix,
		logger,
	)

	dataStore := store.New(
		connectionPool,
		&store.GroupTable{},
		&store.DestinationTable{},
		&store.PolicyTable{},
		conf.TagLength,
	)

	egressDataStore := &store.EgressPolicyStore{
		EgressPolicyRepo: &store.EgressPolicyTable{
			Conn:  connectionPool,
			Guids: &store.GuidGenerator{},
		},
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

	wrappedEgressStore := &store.EgressPolicyMetricsWrapper{
		Store:         egressDataStore,
		MetricsSender: metricsSender,
	}

	errorResponse := &httperror.ErrorResponse{
		MetricsSender: metricsSender,
	}
	payloadValidator := &api.PayloadValidator{PolicyValidator: &api.Validator{}, EgressPolicyValidator: &api.EgressValidator{}}
	policyMapperV0Internal := api_v0_internal.NewMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal))
	policyMapperV1 := api.NewMapper(marshal.UnmarshalFunc(json.Unmarshal), marshal.MarshalFunc(json.Marshal), payloadValidator)

	internalPoliciesHandlerV0 := handlers.NewPoliciesIndexInternal(logger, wrappedStore,
		wrappedEgressStore, policyMapperV0Internal, errorResponse)
	internalPoliciesHandlerV1 := handlers.NewPoliciesIndexInternal(logger, wrappedStore,
		wrappedEgressStore, policyMapperV1, errorResponse)

	createTagsHandlerV1 := &handlers.TagsCreate{
		Store:         wrappedStore,
		ErrorResponse: errorResponse,
	}

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

	err = dropsonde.Initialize(conf.MetronAddress, jobPrefix)
	if err != nil {
		log.Fatalf("%s.%s: initializing dropsonde: %s", logPrefix, jobPrefix, err)
	}

	metricsEmitter := common.InitMetricsEmitter(logger, wrappedStore)

	internalRoutes := rata.Routes{
		{Name: "internal_policies", Method: "GET", Path: "/networking/:version/internal/policies"},
		{Name: "create_tags", Method: "PUT", Path: "/networking/v1/internal/tags"},
	}
	internalHandlers := rata.Handlers{
		"internal_policies": metricsWrap("InternalPolicies", logWrap(
			versionWrap(internalPoliciesHandlerV1, internalPoliciesHandlerV0),
		)),
		"create_tags": metricsWrap("CreateTags", logWrap(createTagsHandlerV1)),
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
	if connectionPool != nil {
		connectionPool.Close()
	}
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}

	logger.Info("exited")
}
