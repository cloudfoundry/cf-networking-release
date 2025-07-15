package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/lagerlevel"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware/adapter"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagerflags"
	"code.cloudfoundry.org/lib/common"
	"code.cloudfoundry.org/service-discovery-controller/addresstable"
	"code.cloudfoundry.org/service-discovery-controller/config"
	"code.cloudfoundry.org/service-discovery-controller/localip"
	"code.cloudfoundry.org/service-discovery-controller/mbus"
	"code.cloudfoundry.org/service-discovery-controller/routes"
	"code.cloudfoundry.org/tlsconfig"
	"github.com/cloudfoundry/dropsonde"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	err := mainWithError()
	if err != nil {
		os.Exit(2)
	}
}

func mainWithError() error {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, os.Interrupt)
	configPath := flag.String("c", "", "path to config file")
	flag.Parse()

	logger, sink := lagerflags.NewFromConfig("service-discovery-controller", common.GetLagerConfig())
	conf, err := readConfig(configPath, logger)
	if err != nil {
		return err
	}

	addressTable := buildAddressTable(conf, logger)

	metronAddress := fmt.Sprintf("127.0.0.1:%d", conf.MetronPort)
	err = dropsonde.Initialize(metronAddress, "service-discovery-controller")
	if err != nil {
		logger.Error("Failed to build subscriber", err)
		return err
	}

	routeMessageRecorder := mbus.NewMetricsRecorder(clock.NewClock())

	subscriber, err := buildSubscriber(conf, addressTable, routeMessageRecorder, logger)
	if err != nil {
		logger.Error("Failed to build subscriber", err)
		return err
	}

	dnsRequestRecorder := &routes.MetricsRecorder{}

	dnsRequestSource := metrics.MetricSource{
		Name:   "dnsRequest",
		Unit:   "request",
		Getter: dnsRequestRecorder.Getter,
	}

	routeMessageSource := metrics.MetricSource{
		Name:   "maxRouteMessageTimePerInterval",
		Unit:   "ms",
		Getter: routeMessageRecorder.GetMaxSinceLastInterval,
	}

	registerMessagesReceivedSource := metrics.MetricSource{
		Name:   "registerMessagesReceived",
		Unit:   "ms",
		Getter: routeMessageRecorder.GetRegisterMessagesReceived,
	}

	metricsEmitter := metrics.NewMetricsEmitter(
		logger,
		time.Duration(conf.MetricsEmitSeconds)*time.Second,
		metrics.NewUptimeSource(),
		dnsRequestSource,
		routeMessageSource,
		registerMessagesReceivedSource,
	)

	metricsSender := &metrics.MetricsSender{
		Logger: logger.Session("time-metric-emitter"),
	}

	logLevelServer := lagerlevel.NewServer(
		conf.LogLevelAddress,
		conf.LogLevelPort,
		time.Duration(conf.ReadHeaderTimeout),
		sink,
		logger.Session("log-level-server"),
	)

	routesServer := routes.NewServer(
		addressTable,
		conf,
		dnsRequestRecorder,
		metricsSender,
		logger.Session("routes-server"),
	)

	members := grouper.Members{
		{Name: "subscriber", Runner: subscriber},
		{Name: "metrics-emitter", Runner: metricsEmitter},
		{Name: "log-level-server", Runner: logLevelServer},
		{Name: "routes-server", Runner: routesServer},
	}

	group := grouper.NewOrdered(os.Interrupt, members)
	monitor := ifrit.Invoke(sigmon.New(group))

	go func() {
		err := <-monitor.Wait()
		if err != nil {
			logger.Fatal("ifrit-failure", err)
		}
	}()

	logger.Info("server-started")

	stopSignal := <-signalChannel
	subscriber.Close()
	addressTable.Shutdown()
	monitor.Signal(stopSignal)
	logger.Info("server-stopped")
	return nil
}

func buildAddressTable(conf *config.Config, logger lager.Logger) *addresstable.AddressTable {
	return addresstable.NewAddressTable(
		time.Duration(conf.StalenessThresholdSeconds)*time.Second,
		time.Duration(conf.PruningIntervalSeconds)*time.Second,
		time.Duration(conf.ResumePruningDelaySeconds)*time.Second,
		clock.NewClock(),
		logger.Session("address-table"))
}

func readConfig(configPath *string, logger lager.Logger) (*config.Config, error) {
	var err error
	bytes, err := os.ReadFile(*configPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Could not read config file at path '%s'", *configPath), err)
		return nil, err
	}
	conf, err := config.NewConfig(bytes)
	if err != nil {
		logger.Error(fmt.Sprintf("Could not parse config file at path '%s'", *configPath), err)
		return nil, err
	}
	return conf, nil
}

func buildSubscriber(conf *config.Config, addressTable *addresstable.AddressTable,
	routeMessageRecorder *mbus.MetricsRecorder, logger lager.Logger) (*mbus.Subscriber, error) {
	uuidGenerator := adapter.UUIDAdapter{}

	uuid, err := uuidGenerator.GenerateUUID()
	if err != nil {
		return &mbus.Subscriber{}, err
	}

	subscriberID := fmt.Sprintf("%s-%s", conf.Index, uuid)

	subOpts := mbus.SubscriberOpts{
		ID:                               subscriberID,
		MinimumRegisterIntervalInSeconds: conf.ResumePruningDelaySeconds,
		PruneThresholdInSeconds:          120,
	}

	provider := &mbus.NatsConnWithUrlProvider{
		Url: strings.Join(conf.NatsServers(), ","),
	}
	if conf.Nats.TLSEnabled {
		provider.TLSConfig, err = tlsconfig.Build(
			tlsconfig.WithInternalServiceDefaults(),
			tlsconfig.WithIdentity(conf.Nats.ClientAuthCertificate),
		).Client(
			tlsconfig.WithAuthority(conf.Nats.CAPool),
		)
		if err != nil {
			return nil, fmt.Errorf("error building TLS config for NATS: %w", err)
		}
	}

	localIP, err := localip.LocalIP()
	if err != nil {
		return &mbus.Subscriber{}, err
	}

	newClock := clock.NewClock()
	warmDuration := time.Duration(conf.WarmDurationSeconds) * time.Second

	subscriber := mbus.NewSubscriber(provider, subOpts, warmDuration, addressTable,
		localIP, routeMessageRecorder, logger.Session("mbus"), newClock)
	return subscriber, nil
}
