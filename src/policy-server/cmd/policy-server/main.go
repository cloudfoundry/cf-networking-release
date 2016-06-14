package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/db"
	"lib/marshal"
	"log"
	"net/http"
	"os"
	"policy-server/config"
	"policy-server/handlers"
	"policy-server/store"
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

	group := &store.Group{}
	destination := &store.Destination{}
	policy := &store.Policy{}

	retriableConnector := db.RetriableConnector{
		Connector:     db.GetConnectionPool,
		Sleeper:       db.SleeperFunc(time.Sleep),
		RetryInterval: 3 * time.Second,
		MaxRetries:    10,
	}

	databaseURL := conf.DatabaseURL
	dbConnectionPool, err := retriableConnector.GetConnectionPool(databaseURL)
	if err != nil {
		log.Fatalf("db connect: %s", err)
	}

	dataStore, err := store.New(dbConnectionPool, group, destination, policy)
	if err != nil {
		log.Fatalf("failed to construct datastore: %s", err)
	}

	unmarshaler := marshal.UnmarshalFunc(json.Unmarshal)

	createPolicyHandler := &handlers.CreatePolicyHandler{
		Store:       dataStore,
		Logger:      lager.NewLogger("policy_server"),
		Unmarshaler: unmarshaler,
	}

	mux := http.NewServeMux()
	mux.Handle("/", uptimeHandler)
	mux.Handle("/networking/v0/external/whoami", whoamiHandler)
	mux.Handle("/networking/v0/external/policies", createPolicyHandler)
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort),
		Handler: mux,
	}

	logger.Info("starting", lager.Data{"listen-address": conf.ListenHost, "port": conf.ListenPort})

	log.Fatal(server.ListenAndServe())
}
