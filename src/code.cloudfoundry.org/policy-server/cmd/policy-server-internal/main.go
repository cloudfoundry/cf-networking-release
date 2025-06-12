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

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/httperror"
	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	middlewareAdapter "code.cloudfoundry.org/cf-networking-helpers/middleware/adapter"
	"code.cloudfoundry.org/cf-networking-helpers/mutualtls"
	"code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagerflags"
	"code.cloudfoundry.org/lib/common"
	"code.cloudfoundry.org/lib/nonmutualtls"
	"code.cloudfoundry.org/policy-server/api"
	"code.cloudfoundry.org/policy-server/cc_client"
	"code.cloudfoundry.org/policy-server/config"
	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/uaa_client"
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

	var uaaTlsConfig *tls.Config
	if conf.SkipSSLValidation {
		uaaTlsConfig = &tls.Config{
			InsecureSkipVerify: conf.SkipSSLValidation,
		}
	} else {
		uaaTlsConfig, err = nonmutualtls.NewClientTLSConfig(conf.UAACA, conf.CCCA)
		if err != nil {
			log.Fatalf("%s.%s error creating tls config: %s", logPrefix, jobPrefix, err) // not tested
		}
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: uaaTlsConfig,
		},
	}

	uaaClient := &uaa_client.Client{
		BaseURL:    fmt.Sprintf("%s:%d", conf.UAAURL, conf.UAAPort),
		Name:       conf.UAAClient,
		Secret:     conf.UAAClientSecret,
		HTTPClient: httpClient,
		Logger:     logger,
	}

	ccClient := &cc_client.Client{
		ExternalJSONClient: json_client.New(logger.Session("cc-json-client"), httpClient, conf.CCURL),
		Logger:             logger,
	}

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
		log.Fatal(err.Error())
	}

	dataStore := store.New(
		connectionPool,
		&store.GroupTable{},
		&store.DestinationTable{},
		&store.PolicyTable{},
		conf.TagLength,
	)

	securityGroupsStore := &store.SGStore{
		Conn:        connectionPool,
		UAAClient:   uaaClient,
		CCClient:    ccClient,
		Logger:      logger,
		CacheExpiry: time.Second * 60,
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

	internalPoliciesLastUpdatedHandlerV1 := handlers.NewPoliciesLastUpdatedInternal(logger, wrappedStore, errorResponse)

	createTagsHandlerV1 := &handlers.TagsCreate{
		Store:         wrappedStore,
		ErrorResponse: errorResponse,
	}

	asgMapper := api.NewAsgMapper(marshal.MarshalFunc(json.Marshal))
	securityGroupsHandlerV1 := handlers.NewAsgsIndex(wrappedSecurityGroupsStore, asgMapper, errorResponse)
	internalSecurityGroupsLastUpdatedHandlerV1 := handlers.NewSecurityGroupsLastUpdatedInternal(logger, wrappedSecurityGroupsStore, errorResponse)

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
		{Name: "internal_policies_last_updated", Method: "GET", Path: "/networking/:version/internal/policies_last_updated"},
		{Name: "internal_security_groups", Method: "GET", Path: "/networking/:version/internal/security_groups"},
		{Name: "internal_security_groups_last_updated", Method: "GET", Path: "/networking/:version/internal/security_groups_last_updated"},
	}

	internalHandlers := rata.Handlers{
		"create_tags":                           metricsWrap("CreateTags", logWrap(createTagsHandlerV1)),
		"internal_policies":                     metricsWrap("InternalPolicies", logWrap(internalPoliciesHandlerV1)),
		"internal_policies_last_updated":        metricsWrap("InternalPoliciesLastUpdated", logWrap(internalPoliciesLastUpdatedHandlerV1)),
		"internal_security_groups":              metricsWrap("InternalSecurityGroups", logWrap(securityGroupsHandlerV1)),
		"internal_security_groups_last_updated": metricsWrap("InternalSecurityGroupsLastUpdated", logWrap(internalSecurityGroupsLastUpdatedHandlerV1)),
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

	healthCheckServer := common.InitServer(logger, nil, "127.0.0.1",
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
		closeErr := connectionPool.Close()
		if closeErr != nil {
			logger.Error("error-closing-connection-pool", err)
		}
	}
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}

	logger.Info("exited")
}
