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
func (c *Client) GetPolicies() ([]models.Policy, error) {
	reqURL := c.url + "/networking/v0/internal/policies"
	request, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("http client do: %s", err)
	}
	defer resp.Body.Close() // untested

	policyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("body read: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("http client do: bad response status %d", resp.StatusCode)
		c.logger.Error("http-client", err, lager.Data{
			"body": string(policyBytes),
		})
		return nil, err
	}

	c.logger.Debug("get-policies", lager.Data{
		"body": string(policyBytes),
	})

	var policies struct {
		Policies []models.Policy `json:"policies"`
	}
	err = c.unmarshaler.Unmarshal(policyBytes, &policies)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal: %s", err)
	}

	return policies.Policies, nil
}
