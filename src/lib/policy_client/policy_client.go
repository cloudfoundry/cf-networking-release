package policy_client

import (
	"encoding/json"
	"lib/marshal"
	"lib/models"
	"net/http"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/http_client.go --fake-name HTTPClient . httpClient
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

//go:generate counterfeiter -o ../fakes/json_client.go --fake-name JSONClient . jsonClient
type jsonClient interface {
	Do(method, route string, reqData, respData interface{}) error
}

type Client struct {
	JsonClient jsonClient
}

func New(logger lager.Logger, httpClient httpClient, url string) *Client {
	return &Client{
		JsonClient: &JsonClient{
			Logger:      logger,
			HttpClient:  httpClient,
			Url:         url,
			Marshaler:   marshal.MarshalFunc(json.Marshal),
			Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
		},
	}
}

func (c *Client) GetPolicies() ([]models.Policy, error) {
	var policies struct {
		Policies []models.Policy `json:"policies"`
	}
	err := c.JsonClient.Do("GET", "/networking/v0/internal/policies", nil, &policies)
	if err != nil {
		return nil, err
	}
	return policies.Policies, nil
}

func (c *Client) AddPolicies(policies []models.Policy) error {
	err := c.JsonClient.Do("POST", "/networking/v0/external/policies", policies, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) DeletePolicies(policies []models.Policy) error {
	err := c.JsonClient.Do("DELETE", "/networking/v0/external/policies", policies, nil)
	if err != nil {
		return err
	}
	return nil
}
