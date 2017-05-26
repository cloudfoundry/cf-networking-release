package main

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/lager"
)

func main() {
	logger := lager.NewLogger("test")
	proxy2Client := json_client.New(logger, http.DefaultClient, "https://proxy-2.cfapps.io")

	nInstances := 100
	overlayIPs := make(map[string]struct{})
	requests := 0

	for {
		ip, err := overlayIP(proxy2Client)
		if err != nil {
			panic(err)
		}
		requests++
		overlayIPs[ip] = struct{}{}
		if len(overlayIPs) == nInstances {
			break
		}
	}

	for ip, _ := range overlayIPs {
		fmt.Printf("%s:8080\n", ip)
	}
}

func overlayIP(jsonClient json_client.JsonClient) (string, error) {
	var resp struct{ ListenAddresses []string }
	err := jsonClient.Do("GET", "/", nil, &resp, "")
	if err != nil {
		return "", err
	}
	return resp.ListenAddresses[1], nil
}
