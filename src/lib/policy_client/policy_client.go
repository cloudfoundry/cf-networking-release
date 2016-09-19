package policy_client

import (
	"fmt"
	"io/ioutil"
	"lib/marshal"
	"lib/models"
	"net/http"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/http_client.go --fake-name HTTPClient . httpClient
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	logger      lager.Logger
	httpClient  httpClient
	url         string
	unmarshaler marshal.Unmarshaler
}

func New(logger lager.Logger, httpClient httpClient, url string, unmarshaler marshal.Unmarshaler) *Client {
	return &Client{
		logger:      logger,
		httpClient:  httpClient,
		url:         url,
		unmarshaler: unmarshaler,
	}
}

func (c *Client) do(method, route string, reqData, respData interface{}) error {
	reqURL := c.url + route
	request, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("http client do: %s", err)
	}
	defer resp.Body.Close() // untested

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("body read: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("http client do: bad response status %d", resp.StatusCode)
		c.logger.Error("http-client", err, lager.Data{
			"body": string(respBytes),
		})
		return err
	}

	c.logger.Debug("http-do", lager.Data{
		"body": string(respBytes),
	})

	if respData != nil {
		err = c.unmarshaler.Unmarshal(respBytes, &respData)
		if err != nil {
			return fmt.Errorf("json unmarshal: %s", err)
		}
	}

	return nil
}

func (c *Client) GetPolicies() ([]models.Policy, error) {
	var policies struct {
		Policies []models.Policy `json:"policies"`
	}
	err := c.do("GET", "/networking/v0/internal/policies", nil, &policies)
	if err != nil {
		return nil, err
	}
	return policies.Policies, nil
}
