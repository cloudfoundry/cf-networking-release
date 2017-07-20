package main

import (
	"log"
	"net/http"
	"time"
	"flag"
	"fmt"
)

func main() {
	if err := mainWithError(); err != nil {
		log.Fatalf("silk-daemon-teardown: %s", err)
	}
}

func pingServer(url, hostType string) (int, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{},
		Timeout: 5 * time.Second,
	}

	response, err := httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("pinging %s failed with: %s", hostType, err)
	}

	return response.StatusCode, err
}

func mainWithError() error {
	repUrl := flag.String("repUrl", "", "path to rep url")
	silkDaemonUrl := flag.String("silkDaemonUrl", "", "path to silk daemon url")
	flag.Parse()

	_, err := pingServer(*repUrl, "rep")
	if err != nil {
		return err
	}

	_, err = pingServer(*silkDaemonUrl, "silk-daemon")
	if err != nil {
		return err
	}

	return nil
}
