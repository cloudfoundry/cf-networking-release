package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
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

func main() {
	conf := &config.Netmon{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	logger := lager.NewLogger("netmon")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatal("error reading config")
	}

	err = json.Unmarshal(configBytes, conf)
	if err != nil {
		log.Fatal("error unmarshalling config")
	}
	logger.Info("parsed-config", lager.Data{"config": conf})

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
