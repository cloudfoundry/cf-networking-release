package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"policy-server/config"
)

type handler struct{}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
}

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

	handler := &handler{}
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", conf.ListenHost, conf.ListenPort),
		Handler: handler,
	}

	fmt.Printf("starting server at %s:%d\n", conf.ListenHost, conf.ListenPort)

	log.Fatal(server.ListenAndServe())
}
