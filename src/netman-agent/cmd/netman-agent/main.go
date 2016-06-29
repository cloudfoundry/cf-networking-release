package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"netman-agent/config"
	"os"
	"time"

	"github.com/pivotal-golang/lager"
)

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

	for {
		url := fmt.Sprintf("%s/internal/v0/policies", conf.PolicyServerURL)
		resp, err := http.Get(url)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		logger.Info("got-policies", lager.Data{"response-body": string(bodyBytes)})

		time.Sleep(pollInterval)
	}
}
