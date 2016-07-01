package client

import (
	"bytes"
	"fmt"
	"lib/marshal"
	"net"
	"net/http"
)

//go:generate counterfeiter -o ../fakes/http_client.go --fake-name HTTPClient . httpClient
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

func New(httpClient httpClient, baseURL string, marshaler marshal.Marshaler) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		marshaler:  marshaler,
	}
}

type Client struct {
	baseURL    string
	httpClient httpClient
	marshaler  marshal.Marshaler
}

func (c *Client) send(method string, data interface{}) error {
	reqURL := c.baseURL + "/cni_result"
	reqBytes, err := c.marshaler.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}

	request, err := http.NewRequest(method, reqURL, bytes.NewReader(reqBytes))
	if err != nil {
		return fmt.Errorf("constructing request: %s", err)
	}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("http client do: %s", err)
	}
	defer resp.Body.Close() // untested

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code %d", resp.StatusCode)
		return err
	}

	return nil
}

func (c *Client) Add(containerID string, groupID string, containerIP net.IP) error {
	return c.send("POST", struct {
		ContainerID string `json:"container_id"`
		GroupID     string `json:"group_id"`
		IP          string `json:"ip"`
	}{
		ContainerID: containerID,
		GroupID:     groupID,
		IP:          containerIP.String(),
	})
}

func (c *Client) Del(containerID string) error {
	return c.send("DELETE", struct {
		ContainerID string `json:"container_id"`
	}{
		ContainerID: containerID,
	})
}
