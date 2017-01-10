package policy_client

import (
	"lib/json_client"
	"lib/models"

	"code.cloudfoundry.org/lager"
)

type InternalClient struct {
	JsonClient json_client.JsonClient
}

func NewInternal(logger lager.Logger, httpClient json_client.HttpClient, baseURL string) *InternalClient {
	return &InternalClient{
		JsonClient: json_client.New(logger, httpClient, baseURL),
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
