package main

import (
	"log"
	"net/http"
	"time"
	"flag"
	"fmt"
	"errors"
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
	repTimeout := flag.Int("repTimeout", 5, "timeout inbetween calls to rep")

	silkDaemonUrl := flag.String("silkDaemonUrl", "", "path to silk daemon url")
	silkDaemonTimeout := flag.Int("silkDaemonTimeout", 2, "timeout inbetween calls to silk daemon")

	flag.Parse()

	var err error
	response := 200
	for response == 200 {
		response, err = pingServer(*repUrl, "rep")
		fmt.Println( fmt.Sprintf("%s: waiting for the rep to exit", time.Now()))
		if err != nil {
			return err
		}
		time.Sleep(time.Duration(*repTimeout) * time.Second)
	}

	response = 200
	numberOfTimesSilkDaemonServerHit := 0
	for response == 200 && numberOfTimesSilkDaemonServerHit < 5 {
		response, err = pingServer(*silkDaemonUrl, "silk-daemon")
		fmt.Println( fmt.Sprintf("%s: waiting for the silk daemon to exit", time.Now()))
		if err != nil {
			return err
		}
		time.Sleep(time.Duration(*silkDaemonTimeout) * time.Second)
		numberOfTimesSilkDaemonServerHit++
	}

	if didSilkDaemonServerExit(numberOfTimesSilkDaemonServerHit) {
		return errors.New("Silk Daemon Server did not exit after 5 ping attempts")
	}

	return nil
}
func didSilkDaemonServerExit(numberOfTimesSilkDaemonServerHit int) bool {
	return numberOfTimesSilkDaemonServerHit == 5
}
