package main

import (
	"encoding/json"
	"fmt"
	"lib/marshal"
	"lib/policy_client"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
)

func main() {
	logger := lager.NewLogger("test")
	jsonClient := policy_client.JsonClient{
		Logger:      logger,
		Url:         "https://c2c-test.cfapps.io",
		HttpClient:  http.DefaultClient,
		Marshaler:   marshal.MarshalFunc(json.Marshal),
		Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
	}

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

func sampleOne(jsonClient policy_client.JsonClient) (time.Duration, bool, error) {
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
