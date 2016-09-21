package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
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

func main() {
	conf := &config.Netmon{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	logger := lager.NewLogger("netmon")
	sink := lager.NewReconfigurableSink(lager.NewWriterSink(os.Stdout, lager.DEBUG), lager.DEBUG)
	logger.RegisterSink(sink)

	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		logger.Fatal("reading config", err)
	}

	err = json.Unmarshal(configBytes, conf)
	if err != nil {
		logger.Fatal("unmarshaling config", err)
	}
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
