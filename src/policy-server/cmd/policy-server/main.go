package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"policy-server/config"
	"policy-server/handlers"
	"policy-server/uaa_client"
	"time"
)

func main() {
	conf := &config.Config{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	configData, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatal("error reading config")
	}

	err = json.Unmarshal(configData, conf)
	if err != nil {
		log.Fatal("error unmarshalling config")
	}

	uaaRequestClient := &uaa_client.Client{
		Host:       conf.UAAURL,
		Name:       conf.UAAClient,
		Secret:     conf.UAAClientSecret,
		HTTPClient: http.DefaultClient,
	}
	whoamiHandler := &handlers.WhoAmIHandler{
		Client: uaaRequestClient,
	}
	uptimeHandler := &handlers.UptimeHandler{
		StartTime: time.Now(),
	}
	mux := http.NewServeMux()
	mux.Handle("/", uptimeHandler)
	mux.Handle("/networking/v0/external/whoami", whoamiHandler)
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort),
		Handler: mux,
	}

	fmt.Printf("starting server at %s:%d\n", conf.ListenHost, conf.ListenPort)

	log.Fatal(server.ListenAndServe())
}
