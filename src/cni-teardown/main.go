package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/silk/client/config"
)

func main() {
	if err := mainWithError(); err != nil {
		log.Fatalf("silk-teardown error: %s", err)
	}
}

func mainWithError() error {
	configFilePath := flag.String("config", "", "path to config file")
	flag.Parse()
	cfg, err := config.LoadConfig(*configFilePath)
	if err != nil {
		return fmt.Errorf("load config file: %s", err)
	}

	logger := lager.NewLogger(fmt.Sprintf("%s.%s", cfg.LogPrefix, "silk-teardown"))
	sink := lager.NewWriterSink(os.Stdout, lager.INFO)
	logger.RegisterSink(sink)

	logger.Info("starting")

	//remove ifb devices
	// get all devices, check if ifb and starts with "i-" and delete if so

	// remove directories

	var errList error
	logger.Info("complete")

	return errList
}
