package main

import (
	"flag"
	"fmt"
	"log"
	"netmon/config"
	"netmon/poller"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/dropsonde"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

var (
	logPrefix = "cfnetworking"
)

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()
	conf, err := config.New(*configFilePath)
	if err != nil {
		log.Fatalf("%s.netmon: reading config: %s", logPrefix, err)
	}

	if conf.LogPrefix != "" {
		logPrefix = conf.LogPrefix
	}

	logger := lager.NewLogger(fmt.Sprintf("%s.netmon", logPrefix))
	sink := lager.NewReconfigurableSink(lager.NewWriterSink(os.Stdout, lager.DEBUG), lager.DEBUG)
	logger.RegisterSink(sink)
	logger.Info("parsed-config", lager.Data{"config": conf})

	logLevel, err := conf.ParseLogLevel()
	if err != nil {
		logger.Fatal("parsing-log-level", err)
	}

	sink.SetMinLevel(logLevel)

	pollInterval := time.Duration(conf.PollInterval) * time.Second
	if pollInterval == 0 {
		pollInterval = time.Second
	}

	dropsonde.Initialize(conf.MetronAddress, "netmon")
	systemMetrics := &poller.SystemMetrics{
		Logger:        logger,
		PollInterval:  pollInterval,
		InterfaceName: conf.InterfaceName,
	}

	members := grouper.Members{
		{"metric_poller", systemMetrics},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("starting")
	err = <-monitor.Wait()
	if err != nil {
		logger.Fatal("ifrit monitor", err)
	}
}
