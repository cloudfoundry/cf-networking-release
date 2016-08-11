package client

import (
	"encoding/json"
	"example-apps/reflex/models"
	"fmt"
	"net/http"
)

//go:generate counterfeiter -o ../fakes/http_client.go --fake-name HttpClient . httpClient
type httpClient interface {
	Get(url string) (*http.Response, error)
}

type ReflexClient struct {
	HttpClient httpClient
	AppURL     string
	AppPort    int
}

func (c *ReflexClient) getPeers(baseURL string) ([]string, error) {
	resp, err := c.HttpClient.Get(fmt.Sprintf("http://%s/peers", baseURL))
	if err != nil {
		return []string{}, err
	}

	var respData models.PeersResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return []string{}, err
	}

	return respData.IPs, nil
}

func (c *ReflexClient) GetAddressesViaRouter() ([]string, error) {
	return c.getPeers(c.AppURL)
}

func (c *ReflexClient) CheckInstance(address string) bool {
	ips, err := c.getPeers(fmt.Sprintf("%s:%d", address, c.AppPort))
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip == address {
			return true
		}
	}
	return false
}
