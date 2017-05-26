package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/lager"
)

func main() {
	logger := lager.NewLogger("test")
	jsonClient := json_client.New(logger, http.DefaultClient, "https://c2c-test.cfapps.io")

	nSamples := 1000

	sampleOne(jsonClient)

	netmanSamples := []time.Duration{}
	kawasakiSamples := []time.Duration{}

	for i := 0; i < nSamples; i++ {
		duration, isNetman, err := sampleOne(jsonClient)
		if err != nil {
			panic(err)
		}
		if isNetman {
			netmanSamples = append(netmanSamples, duration)
		} else {
			kawasakiSamples = append(kawasakiSamples, duration)
		}
	}

	report(netmanSamples, "NETMAN")
	report(kawasakiSamples, "KAWASAKI")
}

func sampleOne(jsonClient json_client.JsonClient) (time.Duration, bool, error) {
	startTime := time.Now()
	var resp struct{ ListenAddresses []string }
	err := jsonClient.Do("GET", "/", nil, &resp, "")
	if err != nil {
		return time.Duration(0), false, err
	}
	isNetman := strings.Contains(resp.ListenAddresses[1], "10.255")
	return time.Since(startTime), isNetman, nil
}

func report(samples []time.Duration, name string) {
	fmt.Println("")
	fmt.Println(name)
	fmt.Println()

	for _, sample := range samples {
		fmt.Printf("%f\n", sample.Seconds())
	}
}
