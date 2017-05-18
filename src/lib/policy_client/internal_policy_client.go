package policy_client

import (
	"errors"
	"lib/models"
	"strings"

	"code.cloudfoundry.org/go-db-helpers/json_client"
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

func (c *InternalClient) GetPoliciesByID(ids ...string) ([]models.Policy, error) {
	var policies struct {
		Policies []models.Policy `json:"policies"`
	}
	if len(ids) == 0 {
		return nil, errors.New("ids cannot be empty")
	}
	err := c.JsonClient.Do("GET", "/networking/v0/internal/policies?id="+strings.Join(ids, ","), nil, &policies, "")
	if err != nil {
		return nil, err
	}
	return policies.Policies, nil
}

func (c *InternalClient) HealthCheck() (bool, error) {
	var healthcheck struct {
		Healthcheck bool `json:"healthcheck"`
	}
	err := c.JsonClient.Do("GET", "/networking/v0/internal/healthcheck", nil, &healthcheck, "")
	if err != nil {
		return false, err
	}
	return healthcheck.Healthcheck, nil
}  
