package policy_client

import (
	"encoding/json"
	"lib/marshal"
	"lib/models"

	"code.cloudfoundry.org/lager"
)

type InternalClient struct {
	JsonClient jsonClient
}

func NewInternal(logger lager.Logger, httpClient httpClient, url string) *InternalClient {
	return &InternalClient{
		JsonClient: &JsonClient{
			Logger:      logger,
			HttpClient:  httpClient,
			Url:         url,
			Marshaler:   marshal.MarshalFunc(json.Marshal),
			Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
		},
	}
}

func (c *InternalClient) GetPolicies() ([]models.Policy, error) {
	var policies struct {
		Policies []models.Policy `json:"policies"`
	}
	err := c.JsonClient.Do("GET", "/networking/v0/internal/policies", nil, &policies, "")
	if err != nil {
		return nil, err
	}
	return policies.Policies, nil
}
