package cc_client

//go:generate counterfeiter -generate

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/lager"
)

const securityGroupsPerPage = 5000

//counterfeiter:generate -o fakes/cc_client.go --fake-name CCClient . CCClient
type CCClient interface {
	GetAppSpaces(token string, appGUIDs []string) (map[string]string, error)
	GetSpace(token, spaceGUID string) (*SpaceResponse, error)
	GetSpaceGUIDs(token string, appGUIDs []string) ([]string, error)
	GetSubjectSpace(token, subjectId string, spaces SpaceResponse) (*SpaceResource, error)
	GetSubjectSpaces(token, subjectId string) (map[string]struct{}, error)
	GetLiveAppGUIDs(token string, appGUIDs []string) (map[string]struct{}, error)
	GetLiveSpaceGUIDs(token string, spaceGUIDs []string) (map[string]struct{}, error)
	GetSecurityGroups(token string) ([]SecurityGroupResource, error)
}

type Client struct {
	Logger     lager.Logger
	JSONClient json_client.JsonClient
}

type GetSecurityGroupsResponse struct {
	Pagination struct {
		TotalPages int `json:"total_pages"`
		First      struct {
			Href string `json:"href"`
		} `json:"first"`
		Last struct {
			Href string `json:"href"`
		} `json:"last"`
		Next struct {
			Href string `json:"href"`
		} `json:"next"`
	} `json:"pagination"`
	Resources []SecurityGroupResource `json:"resources"`
}

type SecurityGroupGloballyEnabled struct {
	Running bool `json:"running"`
	Staging bool `json:"staging"`
}
type SecurityGroupRule struct {
	Protocol    string `json:"protocol"`
	Destination string `json:"destination"`
	Ports       string `json:"ports"`
	Type        int    `json:"type"`
	Code        int    `json:"code"`
	Description string `json:"description"`
	Log         bool   `json:"log"`
}
type SecurityGroupRelationships struct {
	StagingSpaces SecurityGroupSpaceRelationship `json:"staging_spaces"`
	RunningSpaces SecurityGroupSpaceRelationship `json:"running_spaces"`
}
type SecurityGroupSpaceRelationship struct {
	Data []map[string]string `json:"data"`
}
type SecurityGroupResource struct {
	GUID            string                       `json:"guid"`
	Name            string                       `json:"name"`
	GloballyEnabled SecurityGroupGloballyEnabled `json:"globally_enabled"`
	Rules           []SecurityGroupRule          `json:"rules"`
	Relationships   SecurityGroupRelationships   `json:"relationships"`
}

type AppsV3Response struct {
	Pagination struct {
		TotalPages int `json:"total_pages"`
		First      struct {
			Href string `json:"href"`
		} `json:"first"`
		Last struct {
			Href string `json:"href"`
		} `json:"last"`
		Next struct {
			Href string `json:"href"`
		} `json:"next"`
	} `json:"pagination"`
	Resources []struct {
		GUID  string `json:"guid"`
		Links struct {
			Space struct {
				Href string `json:"href"`
			} `json:"space"`
		} `json:"links"`
	} `json:"resources"`
}

type SpacesV3Response struct {
	Pagination struct {
		TotalPages int `json:"total_pages"`
		First      struct {
			Href string `json:"href"`
		} `json:"first"`
		Last struct {
			Href string `json:"href"`
		} `json:"last"`
		Next struct {
			Href string `json:"href"`
		} `json:"next"`
	} `json:"pagination"`
	Resources []struct {
		GUID string `json:"guid"`
	} `json:"resources"`
}

type SpaceResponse struct {
	Entity SpaceEntity `json:"entity"`
}

type SpaceEntity struct {
	Name             string `json:"name"`
	OrganizationGUID string `json:"organization_guid"`
}

type SpaceResource struct {
	Metadata struct {
		GUID string `json:"guid"`
	}
	Entity SpaceEntity `json:"entity"`
}

type SpacesResponse struct {
	TotalResults int64           `json:"total_results"`
	TotalPages   int64           `json:"total_pages"`
	PrevUrl      string          `json:"prev_url"`
	NextUrl      string          `json:"next_url"`
	Resources    []SpaceResource `json:"resources"`
}

func (c *Client) GetAllAppGUIDs(token string) (map[string]struct{}, error) {
	token = fmt.Sprintf("bearer %s", token)

	set := make(map[string]struct{})
	nextPage := "?"
	for nextPage != "" {
		queryParams := strings.Split(nextPage, "?")[1]
		response, err := c.makeAppsV3Request(queryParams, token)
		if err != nil {
			return nil, err
		}
		for _, resource := range response.Resources {
			set[resource.GUID] = struct{}{}
		}
		nextPage = response.Pagination.Next.Href
	}

	return set, nil
}

func (c *Client) makeAppsV3Request(queryParams, token string) (AppsV3Response, error) {
	route := "/v3/apps"
	if queryParams != "" {
		route = fmt.Sprintf("%s?%s", route, queryParams)
	}
	var response AppsV3Response
	err := c.JSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return AppsV3Response{}, fmt.Errorf("json client do: %s", err)
	}
	return response, nil
}

func (c *Client) GetLiveAppGUIDs(token string, appGUIDs []string) (map[string]struct{}, error) {
	token = fmt.Sprintf("bearer %s", token)

	values := url.Values{}
	values.Add("guids", strings.Join(appGUIDs, ","))
	values.Add("per_page", strconv.Itoa(len(appGUIDs)))

	route := fmt.Sprintf("/v3/apps?%s", values.Encode())

	var response AppsV3Response
	err := c.JSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}

	// TotalPages will never be greater than 1, we are setting per_page equal to size of app_guid list
	if response.Pagination.TotalPages > 1 {
		return nil, fmt.Errorf("pagination support not yet implemented")
	}

	set := make(map[string]struct{})
	for _, r := range response.Resources {
		set[r.GUID] = struct{}{}
	}

	return set, nil
}

func (c *Client) GetLiveSpaceGUIDs(token string, spaceGUIDs []string) (map[string]struct{}, error) {
	token = fmt.Sprintf("bearer %s", token)

	liveSpaceGUIDs := make(map[string]struct{})

	values := url.Values{}
	values.Add("guids", strings.Join(spaceGUIDs, ","))
	// Add +1 incase len is 0 - avoiding a capi error
	values.Add("per_page", strconv.Itoa(len(spaceGUIDs)+1))

	route := fmt.Sprintf("/v3/spaces?%s", values.Encode())
	var response SpacesV3Response
	err := c.JSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}

	// TotalPages will never be greater than 1, we are setting per_page equal to size of space_guids list
	if response.Pagination.TotalPages > 1 {
		return nil, fmt.Errorf("pagination support not yet implemented")
	}
	for _, space := range response.Resources {
		liveSpaceGUIDs[space.GUID] = struct{}{}
	}

	return liveSpaceGUIDs, nil
}

func (c *Client) GetSpaceGUIDs(token string, appGUIDs []string) ([]string, error) {
	mapping, err := c.GetAppSpaces(token, appGUIDs)
	if err != nil {
		return nil, err
	}

	deduplicated := map[string]struct{}{}
	for _, spaceID := range mapping {
		deduplicated[spaceID] = struct{}{}
	}

	ret := []string{}
	for spaceID, _ := range deduplicated {
		ret = append(ret, spaceID)
	}

	return ret, nil
}

func (c *Client) GetAppSpaces(token string, appGUIDs []string) (map[string]string, error) {
	if len(appGUIDs) < 1 {
		return map[string]string{}, nil
	}

	token = fmt.Sprintf("bearer %s", token)

	values := url.Values{}
	values.Add("guids", strings.Join(appGUIDs, ","))
	values.Add("per_page", strconv.Itoa(len(appGUIDs)))

	route := fmt.Sprintf("/v3/apps?%s", values.Encode())

	var response AppsV3Response
	err := c.JSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}

	// TotalPages will never be greater than 1, we are setting per_page equal to size of app_guid list
	if response.Pagination.TotalPages > 1 {
		return nil, fmt.Errorf("pagination support not yet implemented")
	}

	set := make(map[string]string)
	for _, r := range response.Resources {
		href := r.Links.Space.Href
		parts := strings.Split(href, "/")
		appID := r.GUID
		spaceID := parts[len(parts)-1]
		set[appID] = spaceID
	}
	return set, nil
}

func (c *Client) GetSpace(token, spaceGUID string) (*SpaceResponse, error) {
	token = fmt.Sprintf("bearer %s", token)
	route := fmt.Sprintf("/v2/spaces/%s", spaceGUID)

	var response SpaceResponse
	err := c.JSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		typedErr, ok := err.(*json_client.HttpResponseCodeError)
		if !ok {
			return nil, fmt.Errorf("json client do: %s", err)
		}
		if typedErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("json client do: %s", err)
	}

	return &response, nil
}

func (c *Client) GetSubjectSpace(token, subjectId string, space SpaceResponse) (*SpaceResource, error) {
	token = fmt.Sprintf("bearer %s", token)

	values := url.Values{}
	values.Add("q", fmt.Sprintf("developer_guid:%s", subjectId))
	values.Add("q", fmt.Sprintf("name:%s", space.Entity.Name))
	values.Add("q", fmt.Sprintf("organization_guid:%s", space.Entity.OrganizationGUID))

	route := fmt.Sprintf("/v2/spaces?%s", values.Encode())

	var response SpacesResponse
	err := c.JSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}

	numSpaces := len(response.Resources)
	if numSpaces == 0 {
		return nil, nil
	}
	if numSpaces > 1 {
		return nil, fmt.Errorf("found more than one matching space")
	}

	return &response.Resources[0], nil
}

func (c *Client) GetSubjectSpaces(token, subjectId string) (map[string]struct{}, error) {
	const maximumPageSize = "100"
	token = fmt.Sprintf("bearer %s", token)

	values := url.Values{}
	values.Add("results-per-page", maximumPageSize)

	route := fmt.Sprintf("/v2/users/%s/spaces?%s", subjectId, values.Encode())

	var resources []SpaceResource
	for route != "" {
		var response SpacesResponse
		err := c.JSONClient.Do("GET", route, nil, &response, token)
		if err != nil {
			return nil, fmt.Errorf("json client do: %s", err)
		}
		route = response.NextUrl
		resources = append(resources, response.Resources...)
	}

	subjectSpaces := map[string]struct{}{}
	for _, space := range resources {
		spaceID := space.Metadata.GUID
		subjectSpaces[spaceID] = struct{}{}
	}

	return subjectSpaces, nil
}

func (c *Client) GetSecurityGroups(token string) ([]SecurityGroupResource, error) {
	token = fmt.Sprintf("bearer %s", token)

	securityGroups := []SecurityGroupResource{}

	nextPage := fmt.Sprintf("?per_page=%d&order_by=created_at", securityGroupsPerPage)
	for nextPage != "" {
		queryParams := strings.Split(nextPage, "?")[1]
		response, err := c.makeGetSecurityGroupsRequest(queryParams, token)
		if err != nil {
			return nil, err
		}
		for _, resource := range response.Resources {
			securityGroups = append(securityGroups, resource)
		}
		nextPage = response.Pagination.Next.Href
	}
	return securityGroups, nil
}

func (c *Client) makeGetSecurityGroupsRequest(queryParams, token string) (*GetSecurityGroupsResponse, error) {
	route := "/v3/security_groups"

	if queryParams != "" {
		route = fmt.Sprintf("%s?%s", route, queryParams)
	}

	var response GetSecurityGroupsResponse
	err := c.JSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}

	return &response, nil
}
