package main

import (
	"flag"
	"log"
	"os"

	"code.cloudfoundry.org/service-discovery-controller/config"
)

var (
	configFile string
)

func main() {
	flag.StringVar(&configFile, "config", "", "Configuration File")
	flag.Parse()

	bytes, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatal("Could not read config file: ", err)
	}

	_, err = config.NewConfig(bytes)
	if err != nil {
		log.Fatal("failed-to-load-config: ", err)
	}
	log.Print("config-loaded-successfully")
}
