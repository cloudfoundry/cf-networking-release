package a8

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

type ServiceInstance struct {
	ServiceName string          `json:"service_name"`
	Endpoint    ServiceEndpoint `json:"endpoint"`
	Status      string          `json:"status,omitempty"`
	TTL         int             `json:"ttl,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
}

type ServiceEndpoint struct {
	Type  string `json:"type"`  // e.g. "http" or "tcp"
	Value string `json:"value"` // e.g. "172.135.10.1:8080" or "http://myapp.bosh-lite.com".
}

type Client struct {
	BaseURL            string
	HttpClient         *http.Client
	LocalServerAddress string
	ServiceName        string
	TTLSeconds         int
}

func (c *Client) createURL(route string) (string, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", fmt.Errorf("unable to parse base url: %s", err)
	}
	u.Path = path.Join(u.Path, route)
	return u.String(), nil
}

func (c *Client) postServiceInstance(serviceInstance ServiceInstance) error {
	reqBytes, err := json.Marshal(serviceInstance)
	if err != nil {
		return err
	}

	url, err := c.createURL("/api/v1/instances")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected response code: %d: %s", resp.StatusCode, respBytes)
	}

	return nil
}

func (c *Client) Register() error {
	serviceInstance := ServiceInstance{
		ServiceName: c.ServiceName,
		Endpoint: ServiceEndpoint{
			Type:  "tcp",
			Value: c.LocalServerAddress,
		},
		Status: "UP",
		TTL:    c.TTLSeconds,
	}

	return c.postServiceInstance(serviceInstance)
}
