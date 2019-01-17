package main

import (
	"bosh-dns-adapter/config"
	"bosh-dns-adapter/handlers"
	"bosh-dns-adapter/sdcclient"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/common"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/lagerlevel"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"
	"github.com/cloudfoundry/dropsonde"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, os.Interrupt)

	logger, reconfigurableSink := lagerflags.NewFromConfig("bosh-dns-adapter", common.GetLagerConfig())
	logger.RegisterSink(lager.NewPrettySink(os.Stderr, lager.ERROR))

	configPath := flag.String("c", "", "path to config file")
	flag.Parse()

	bytes, err := ioutil.ReadFile(*configPath)
	if err != nil {
		logger.Info("Could not read config file", lager.Data{"path": *configPath})
		os.Exit(2)
	}

	config, err := config.NewConfig(bytes)
	if err != nil {
		logger.Info("Could not parse config file", lager.Data{"path": *configPath})
		os.Exit(2)
	}

	address := fmt.Sprintf("%s:%s", config.Address, config.Port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error(fmt.Sprintf("Address (%s) not available", address), err)
		os.Exit(1)
	}

	sdcServerUrl := fmt.Sprintf("https://%s:%s",
		config.ServiceDiscoveryControllerAddress,
		config.ServiceDiscoveryControllerPort,
	)

	metronAddress := fmt.Sprintf("127.0.0.1:%d", config.MetronPort)
	err = dropsonde.Initialize(metronAddress, "bosh-dns-adapter")
	if err != nil {
		logger.Error("Unable to initialize dropsonde", err, lager.Data{"metron_address": metronAddress})
		os.Exit(1)
	}

	sdcClient, err := sdcclient.NewServiceDiscoveryClient(sdcServerUrl, config.CACert, config.ClientCert, config.ClientKey)
	if err != nil {
		logger.Error("Unable to create service discovery client", err)
		os.Exit(1)
	}

	// copilotClient := CopilotClient{}

	metricSender := metrics.MetricsSender{
		Logger: logger.Session("bosh-dns-adapter"),
	}

	metricsWrap := func(name string, handler http.Handler) http.Handler {
		metricsWrapper := middleware.MetricWrapper{
			Name:          name,
			MetricsSender: &metricSender,
		}
		return metricsWrapper.Wrap(handler)
	}

	getIPsHandler := handlers.GetIP{
		SDCClient: sdcClient,
		// CopilotClient:              copilotClient,
		InternalServiceMeshDomains: config.InternalServiceMeshDomains,
		Logger:        logger,
		MetricsSender: &metricSender,
	}

	go func() {
		http.Serve(l, metricsWrap("GetIPs", http.HandlerFunc(getIPsHandler.ServeHTTP)))
	}()

	uptimeSource := metrics.NewUptimeSource()

	metricsEmitter := metrics.NewMetricsEmitter(
		logger,
		time.Duration(config.MetricsEmitSeconds)*time.Second,
		uptimeSource,
	)

	members := grouper.Members{
		{"metrics-emitter", metricsEmitter},
		{"log-level-server", lagerlevel.NewServer(config.LogLevelAddress, config.LogLevelPort, reconfigurableSink, logger.Session("log-level-server"))},
	}
	group := grouper.NewOrdered(os.Interrupt, members)
	monitor := ifrit.Invoke(sigmon.New(group))

	go func() {
		err = <-monitor.Wait()
		if err != nil {
			logger.Error("ifrit-failure", err)
			os.Exit(1)
		}
	}()

	logger.Info("server-started")
	select {
	case sig := <-signalChannel:
		monitor.Signal(sig)
		l.Close()
		logger.Info("server-stopped")
		return
	}
}
