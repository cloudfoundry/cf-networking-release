package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"code.cloudfoundry.org/bosh-dns-adapter/config"
	"code.cloudfoundry.org/bosh-dns-adapter/sdcclient"
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

	config, err := config.NewConfig(bytes)
	if err != nil {
		log.Fatal("failed-to-load-config: ", err)
	}

	sdcServerUrl := fmt.Sprintf("https://%s:%s",
		config.ServiceDiscoveryControllerAddress,
		config.ServiceDiscoveryControllerPort,
	)

	_, err = sdcclient.NewServiceDiscoveryClient(sdcServerUrl, config.CACert, config.ClientCert, config.ClientKey)
	if err != nil {
		log.Fatal("failed-to-create-service-discovery-client: ", err)
	}
	log.Print("config-loaded-successfully")
}
