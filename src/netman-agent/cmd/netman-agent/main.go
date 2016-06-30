package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"lib/marshal"
	"log"
	"net/http"
	"netman-agent/config"
	"netman-agent/models"
	"netman-agent/policy_client"
	"netman-agent/rule_updater"
	"os"
	"time"

	"github.com/pivotal-golang/lager"
)

type fakeStoreReader struct{}

func (r *fakeStoreReader) GetContainers() models.Containers {
	return map[string][]models.Container{
		"app-guid": []models.Container{{
			ID: "some-container-id",
			IP: "8.8.8.8",
		}},
	}
}

func main() {
	conf := &config.Config{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	logger := lager.NewLogger("netman-agent")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatal("error reading config")
	}

	pollInterval := time.Duration(conf.PollInterval) * time.Second
	if pollInterval == 0 {
		pollInterval = time.Second
	}

	err = json.Unmarshal(configBytes, conf)
	if err != nil {
		log.Fatal("error unmarshalling config")
	}

	policyClient := policy_client.New(
		logger.Session("policy-client"),
		http.DefaultClient,
		conf.PolicyServerURL,
		marshal.UnmarshalFunc(json.Unmarshal),
	)
	ruleUpdater := rule_updater.New(
		logger.Session("rules-updater"),
		&fakeStoreReader{},
		policyClient,
	)
	for {
		err = ruleUpdater.Update()
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(pollInterval)
	}
}
