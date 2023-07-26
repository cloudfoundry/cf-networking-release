package policy_client

import (
	"errors"
	"fmt"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/lager/v3"
)

//go:generate counterfeiter -o fakes/internal_policy_client.go --fake-name InternalPolicyClient . InternalPolicyClient
type InternalPolicyClient interface {
	GetPolicies() ([]*Policy, error)
	GetSecurityGroupsForSpace(spaceGuids []string) ([]*SecurityGroup, error)
}

type Config struct {
	PerPageSecurityGroups int
}

var DefaultConfig = Config{
	PerPageSecurityGroups: 5000,
}

type InternalClient struct {
	JsonClient json_client.JsonClient
	Config     Config
}

type TagRequest struct {
	ID   string
	Type string
}

type SecurityGroupsResponse struct {
	Next           int             `json:"next"`
	SecurityGroups []SecurityGroup `json:"security_groups"`
}

func NewInternal(logger lager.Logger, httpClient json_client.HttpClient, baseURL string, conf Config) *InternalClient {
	return &InternalClient{
		JsonClient: json_client.New(logger, httpClient, baseURL),
		Config:     conf,
	}
}

func (c *InternalClient) GetPolicies() ([]*Policy, error) {
	var policies struct {
		Policies []*Policy `json:"policies"`
	}
	err := c.JsonClient.Do("GET", "/networking/v1/internal/policies", nil, &policies, "")
	if err != nil {
		return nil, err
	}
	return policies.Policies, nil
}

func (c *InternalClient) GetPoliciesLastUpdated() (int, error) {
	var lastUpdatedTimestamp int
	err := c.JsonClient.Do("GET", "/networking/v1/internal/policies_last_updated", nil, &lastUpdatedTimestamp, "")
	if err != nil {
		return 0, err
	}
	return lastUpdatedTimestamp, nil
}

func (c *InternalClient) GetPoliciesByID(ids ...string) ([]Policy, error) {
	var policies struct {
		Policies []Policy `json:"policies"`
	}
	if len(ids) == 0 {
		return nil, errors.New("ids cannot be empty")
	}
	err := c.JsonClient.Do("GET", "/networking/v1/internal/policies?id="+strings.Join(ids, ","), nil, &policies, "")
	if err != nil {
		return nil, err
	}
	return policies.Policies, nil
}

func (c *InternalClient) GetSecurityGroupsForSpace(spaceGuids ...string) ([]SecurityGroup, error) {
	var securityGroups []SecurityGroup
	var next int

	for initial := true; initial || next != 0; initial = false {
		url := fmt.Sprintf(
			"/networking/v1/internal/security_groups?per_page=%d",
			c.Config.PerPageSecurityGroups,
		)
		if len(spaceGuids) > 0 {
			url = fmt.Sprintf("%s&space_guids=%s", url, strings.Join(spaceGuids, ","))
		}
		if next != 0 {
			url = fmt.Sprintf("%s&from=%d", url, next)
		}
		var r SecurityGroupsResponse
		err := c.JsonClient.Do("GET", url, nil, &r, "")
		if err != nil {
			return nil, err
		}
		securityGroups = append(securityGroups, r.SecurityGroups...)
		next = r.Next
	}

	return securityGroups, nil
}
func (c *InternalClient) CreateOrGetTag(id, groupType string) (string, error) {
	var response struct {
		ID   string
		Type string
		Tag  string
	}
	err := c.JsonClient.Do("PUT", "/networking/v1/internal/tags", TagRequest{
		ID:   id,
		Type: groupType,
	}, &response, "")
	if err != nil {
		return "", err
	}
	return response.Tag, nil
}

func (c *InternalClient) HealthCheck() (bool, error) {
	var healthcheck struct {
		Healthcheck bool `json:"healthcheck"`
	}
	err := c.JsonClient.Do("GET", "/networking/v1/internal/healthcheck", nil, &healthcheck, "")
	if err != nil {
		return false, err
	}
	return healthcheck.Healthcheck, nil
}
