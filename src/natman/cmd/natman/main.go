package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"natman/config"
	"natman/poller"
	"os"
	"time"

	"code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/garden/client/connection"

	"code.cloudfoundry.org/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	conf := &config.Natman{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	logger := lager.NewLogger("natman")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

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

	gardenClient := client.New(connection.New(conf.GardenProtocol, conf.GardenAddress))
	gardenPoller := &poller.Poller{
		Logger:       logger,
		PollInterval: pollInterval,
		GardenClient: gardenClient,
	}
	members := grouper.Members{
		{"garden_poller", gardenPoller},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("starting")
	err = <-monitor.Wait()
	if err != nil {
		logger.Fatal("ifrit monitor", err)
	}
}
