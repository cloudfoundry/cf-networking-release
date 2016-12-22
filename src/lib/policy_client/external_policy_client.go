package policy_client

import (
	"encoding/json"
	"lib/marshal"
	"lib/models"
	"strings"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/external_policy_client.go --fake-name ExternalPolicyClient . ExternalPolicyClient
type ExternalPolicyClient interface {
	GetPolicies(token string) ([]models.Policy, error)
	GetPoliciesByID(token string, ids ...string) ([]models.Policy, error)
	DeletePolicies(token string, policies []models.Policy) error
	AddPolicies(token string, policies []models.Policy) error
}

type ExternalClient struct {
	JsonClient jsonClient
}

func NewExternal(logger lager.Logger, httpClient httpClient, url string) *ExternalClient {
	return &ExternalClient{
		JsonClient: &JsonClient{
			Logger:      logger,
			HttpClient:  httpClient,
			Url:         url,
			Marshaler:   marshal.MarshalFunc(json.Marshal),
			Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
		},
	}
}

func (c *ExternalClient) GetPolicies(token string) ([]models.Policy, error) {
	var policies struct {
		Policies []models.Policy `json:"policies"`
	}
	err := c.JsonClient.Do("GET", "/networking/v0/external/policies", nil, &policies, token)
	if err != nil {
		return nil, err
	}
	return policies.Policies, nil
}

func (c *ExternalClient) GetPoliciesByID(token string, ids ...string) ([]models.Policy, error) {
	var policies struct {
		Policies []models.Policy `json:"policies"`
	}
	route := "/networking/v0/external/policies?id=" + strings.Join(ids, ",")
	err := c.JsonClient.Do("GET", route, nil, &policies, token)
	if err != nil {
		return nil, err
	}
	return policies.Policies, nil
}

func (c *ExternalClient) AddPolicies(token string, policies []models.Policy) error {
	reqPolicies := map[string][]models.Policy{
		"policies": policies,
	}
	err := c.JsonClient.Do("POST", "/networking/v0/external/policies", reqPolicies, nil, token)
	if err != nil {
		return err
	}
	return nil
}

func (c *ExternalClient) DeletePolicies(token string, policies []models.Policy) error {
	reqPolicies := map[string][]models.Policy{
		"policies": policies,
	}
	err := c.JsonClient.Do("POST", "/networking/v0/external/policies/delete", reqPolicies, nil, token)
	if err != nil {
		return err
	}
	return nil
}
