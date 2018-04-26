package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"service-discovery-controller/addresstable"
	"service-discovery-controller/config"
	"service-discovery-controller/mbus"
	"syscall"
	"time"

	"service-discovery-controller/localip"
	"strings"

	"service-discovery-controller/routes"

	"code.cloudfoundry.org/cf-networking-helpers/lagerlevel"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware/adapter"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
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

	logger, sink := buildLogger()
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

	metricsEmitter := metrics.NewMetricsEmitter(
		logger,
		time.Duration(conf.MetricsEmitSeconds)*time.Second,
		metrics.NewUptimeSource(),
		dnsRequestSource,
		routeMessageSource,
	)

	metricsSender := &metrics.MetricsSender{
		Logger: logger.Session("time-metric-emitter"),
	}

	logLevelServer := lagerlevel.NewServer(
		conf.LogLevelAddress,
		conf.LogLevelPort,
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
		{"subscriber", subscriber},
		{"metrics-emitter", metricsEmitter},
		{"log-level-server", logLevelServer},
		{"routes-server", routesServer},
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

	select {
	case signal := <-signalChannel:
		subscriber.Close()
		addressTable.Shutdown()
		monitor.Signal(signal)
		logger.Info("server-stopped")
		return nil
	}
}

func buildAddressTable(conf *config.Config, logger lager.Logger) *addresstable.AddressTable {
	return addresstable.NewAddressTable(
		time.Duration(conf.StalenessThresholdSeconds)*time.Second,
		time.Duration(conf.PruningIntervalSeconds)*time.Second,
		time.Duration(conf.ResumePruningDelaySeconds)*time.Second,
		clock.NewClock(),
		logger.Session("address-table"))
}

func buildLogger() (lager.Logger, *lager.ReconfigurableSink) {
	logger := lager.NewLogger("service-discovery-controller")
	writerSink := lager.NewWriterSink(os.Stdout, lager.DEBUG)
	sink := lager.NewReconfigurableSink(writerSink, lager.INFO)
	logger.RegisterSink(sink)
	return logger, sink
}

func readConfig(configPath *string, logger lager.Logger) (*config.Config, error) {
	var err error
	bytes, err := ioutil.ReadFile(*configPath)
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
		ID: subscriberID,
		MinimumRegisterIntervalInSeconds: conf.ResumePruningDelaySeconds,
		PruneThresholdInSeconds:          120,
	}

	provider := &mbus.NatsConnWithUrlProvider{
		Url: strings.Join(conf.NatsServers(), ","),
	}

	localIP, err := localip.LocalIP()
	if err != nil {
		return &mbus.Subscriber{}, err
	}

	metricsSender := &metrics.MetricsSender{
		Logger: logger.Session("metrics"),
	}

	clock := clock.NewClock()
	warmDuration := time.Duration(conf.WarmDurationSeconds) * time.Second

	subscriber := mbus.NewSubscriber(provider, subOpts, warmDuration, addressTable,
		localIP, routeMessageRecorder, logger.Session("mbus"), metricsSender, clock)
	return subscriber, nil
}
