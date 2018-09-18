package psclient

import (
	"fmt"

	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/lager"
)

type Client struct {
	JsonClient json_client.JsonClient
}

type IPRange struct {
	Start string
	End   string
}

type Port struct {
	Start int
	End   int
}

type Destination struct {
	GUID     string `json:"id,omitempty"`
	Protocol string
	IPs      []IPRange
	Ports    []Port
}

type DestinationList struct {
	Destinations []Destination
}

type EgressPolicy struct {
	GUID        string                  `json:"id,omitempty"`
	Source      EgressPolicySource      `json:"source"`
	Destination EgressPolicyDestination `json:"destination"`
}

type EgressPolicySource struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id"`
}

type EgressPolicyDestination struct {
	ID string `json:"id"`
}

type EgressPolicyList struct {
	TotalEgressPolicies int            `json:"total_egress_policies,omitempty"`
	EgressPolicies      []EgressPolicy `json:"egress_policies"`
}

func NewClient(logger lager.Logger, httpClient json_client.HttpClient, baseURL string) *Client {
	return &Client{
		JsonClient: json_client.New(logger, httpClient, baseURL),
	}
}

func (c *Client) CreateDestination(destination Destination, token string) (string, error) {
	var response DestinationList
	err := c.JsonClient.Do("POST", "/networking/v1/external/destinations", DestinationList{
		Destinations: []Destination{
			destination,
		},
	}, &response, "Bearer " + token)
	if err != nil {
		return "", fmt.Errorf("json client do: %s", err)
	}

	return response.Destinations[0].GUID, nil
}

func (c *Client) CreateEgressPolicy(egressPolicy EgressPolicy, token string) (string, error) {
	var response EgressPolicyList
	err := c.JsonClient.Do("POST", "/networking/v1/external/egress_policies", EgressPolicyList{
		EgressPolicies: []EgressPolicy{
			egressPolicy,
		},
	}, &response, "Bearer " + token)
	if err != nil {
		return "", fmt.Errorf("json client do: %s", err)
	}

	return response.EgressPolicies[0].GUID, nil
}

func (c *Client) ListEgressPolicies(token string) (EgressPolicyList, error) {
	var response EgressPolicyList
	err := c.JsonClient.Do("GET", "/networking/v1/external/egress_policies", "", &response, "Bearer " + token)
	if err != nil {
		return EgressPolicyList{}, fmt.Errorf("list egress policies api call: %s", err)
	}

	return response, nil
}
