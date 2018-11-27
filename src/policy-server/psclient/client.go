package psclient

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

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
	GUID        string `json:"id,omitempty"`
	Protocol    string
	IPs         []IPRange
	Ports       []Port
	Name        string `json:"name"`
	Description string `json:"description"`
	ICMPType    *int   `json:"icmp_type,omitempty"`
	ICMPCode    *int   `json:"icmp_code,omitempty"`
}

type DestinationList struct {
	Destinations []Destination
}

type ListDestinationsOptions struct {
	QueryNames []string
	QueryIDs   []string
}

type EgressPolicy struct {
	GUID         string             `json:"id,omitempty"`
	Source       EgressPolicySource `json:"source"`
	Destination  Destination        `json:"destination"`
	AppLifecycle string             `json:"app_lifecycle"`
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

func (c *Client) ListDestinations(token string, options ListDestinationsOptions) ([]Destination, error) {
	var response DestinationList
	query := url.Values{}
	if len(options.QueryNames) > 0 {
		query.Set("name", strings.Join(options.QueryNames, ","))
	}
	if len(options.QueryIDs) > 0 {
		query.Set("id", strings.Join(options.QueryIDs, ","))
	}
	url := fmt.Sprintf("/networking/v1/external/destinations?%s", query.Encode())
	err := c.JsonClient.Do("GET", url, nil, &response, "Bearer "+token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}
	return response.Destinations, nil
}

func (c *Client) UpdateDestinations(token string, destinations ...Destination) ([]Destination, error) {
	if len(destinations) == 0 {
		return []Destination{}, errors.New("destinations to be updated must not be empty")
	}
	for _, destination := range destinations {
		if destination.GUID == "" {
			return []Destination{}, errors.New("destinations to be updated must have an ID")
		}
	}

	var response DestinationList
	err := c.JsonClient.Do("PUT", "/networking/v1/external/destinations", DestinationList{
		Destinations: destinations,
	}, &response, "Bearer "+token)

	if err != nil {
		return []Destination{}, fmt.Errorf("json client do: %s", err)
	}
	return response.Destinations, nil
}

func (c *Client) CreateDestinations(token string, destinations ...Destination) ([]Destination, error) {
	var response DestinationList
	err := c.JsonClient.Do("POST", "/networking/v1/external/destinations", DestinationList{
		Destinations: destinations,
	}, &response, "Bearer "+token)
	if err != nil {
		return []Destination{}, fmt.Errorf("json client do: %s", err)
	}

	return response.Destinations, nil
}

func (c *Client) DeleteDestination(token string, destination Destination) ([]Destination, error) {
	var response DestinationList
	err := c.JsonClient.Do("DELETE", "/networking/v1/external/destinations/"+destination.GUID, nil, &response, "Bearer "+token)
	if err != nil {
		return []Destination{}, fmt.Errorf("json client do: %s", err)
	}
	return response.Destinations, nil
}

func (c *Client) CreateEgressPolicy(egressPolicy EgressPolicy, token string) (string, error) {
	var response EgressPolicyList
	err := c.JsonClient.Do("POST", "/networking/v1/external/egress_policies", EgressPolicyList{
		EgressPolicies: []EgressPolicy{
			egressPolicy,
		},
	}, &response, "Bearer "+token)
	if err != nil {
		return "", fmt.Errorf("json client do: %s", err)
	}

	return response.EgressPolicies[0].GUID, nil
}

func (c *Client) DeleteEgressPolicy(egressPolicyGUID, token string) (EgressPolicy, error) {
	var response EgressPolicyList
	err := c.JsonClient.Do("DELETE", fmt.Sprintf("/networking/v1/external/egress_policies/%s", egressPolicyGUID), "", &response, "Bearer "+token)
	if err != nil {
		return EgressPolicy{}, fmt.Errorf("json client do: %s", err)
	}

	return response.EgressPolicies[0], nil
}

func (c *Client) ListEgressPolicies(token string, sourceIDs, sourceTypes, destinationIDs, destinationNames []string) (EgressPolicyList, error) {
	var response EgressPolicyList

	var filter = strings.Join([]string{
		"SourceIDs=" + strings.Join(sourceIDs, ","),
		"SourceTypes=" + strings.Join(sourceTypes, ","),
		"DestinationIDs=" + strings.Join(destinationIDs, ","),
		"DestinationNames=" + strings.Join(destinationNames, ","),
	}, "&")

	err := c.JsonClient.Do("GET", "/networking/v1/external/egress_policies?"+filter, "", &response, "Bearer "+token)
	if err != nil {
		return EgressPolicyList{}, fmt.Errorf("list egress policies api call: %s", err)
	}

	return response, nil
}
