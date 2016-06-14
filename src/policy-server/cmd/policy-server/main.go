package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/marshal"
	"log"
	"net/http"
	"os"
	"policy-server/config"
	"policy-server/handlers"
	"policy-server/uaa_client"
	"time"

	"github.com/pivotal-golang/lager"
)

func main() {
	conf := &config.Config{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	logger := lager.NewLogger("policy-server")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	configData, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatal("error reading config")
	}

	err = json.Unmarshal(configData, conf)
	if err != nil {
		log.Fatal("error unmarshalling config")
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: conf.SkipSSLValidation,
			},
		},
	}

	uaaRequestClient := &uaa_client.Client{
		Host:       conf.UAAURL,
		Name:       conf.UAAClient,
		Secret:     conf.UAAClientSecret,
		HTTPClient: httpClient,
		Logger:     logger,
	}
	whoamiHandler := &handlers.WhoAmIHandler{
		Client:    uaaRequestClient,
		Logger:    logger.Session("external"),
		Marshaler: marshal.MarshalFunc(json.Marshal),
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

	logger.Info("starting", lager.Data{"listen-address": conf.ListenHost, "port": conf.ListenPort})

	log.Fatal(server.ListenAndServe())
}
