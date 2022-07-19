package cc_client

//go:generate counterfeiter -generate

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/json_client"
	"code.cloudfoundry.org/lager"
)

const SECURITY_GROUPS_PER_PAGE = 5000

//counterfeiter:generate -o fakes/cc_client.go --fake-name CCClient . CCClient
type CCClient interface {
	GetAppSpaces(token string, appGUIDs []string) (map[string]string, error)
	GetSpace(token, spaceGUID string) (*SpaceResponse, error)
	GetSpaceGUIDs(token string, appGUIDs []string) ([]string, error)
	GetSubjectSpace(token, subjectId string, spaces SpaceResponse) (*SpaceResource, error)
	GetSubjectSpaces(token, subjectId string) (map[string]struct{}, error)
	GetLiveAppGUIDs(token string, appGUIDs []string) (map[string]struct{}, error)
	GetLiveSpaceGUIDs(token string, spaceGUIDs []string) (map[string]struct{}, error)
	GetSecurityGroupsLastUpdate(token string) (time.Time, error)
	GetSecurityGroupsWithPage(token string, page int) (GetSecurityGroupsResponse, error)
	GetSecurityGroups(token string) ([]SecurityGroupResource, error)
}

type Client struct {
	Logger             lager.Logger
	ExternalJSONClient json_client.JsonClient
	InternalJSONClient json_client.JsonClient
}

type Href struct {
	Href string `json:"href"`
}

type Pagination struct {
	TotalPages int  `json:"total_pages"`
	First      Href `json:"first"`
	Last       Href `json:"last"`
	Next       Href `json:"next"`
	Previous   Href `json:"previous"`
}

type GetSecurityGroupsResponse struct {
	Pagination Pagination              `json:"pagination"`
	Resources  []SecurityGroupResource `json:"resources"`
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

type SecurityGroupLatestUpdateResponse struct {
	LastUpdate string `json:"last_update"`
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
	c.Logger.Info("get-all-app-guids")
	token = fmt.Sprintf("bearer %s", token)

	set := make(map[string]struct{})
	nextPage := "?"
	for nextPage != "" {
		queryParams := strings.Split(nextPage, "?")[1]
		c.Logger.Debug("make-apps-v3-request", lager.Data{"queryParams": queryParams})
		response, err := c.makeAppsV3Request(queryParams, token)
		if err != nil {
			return nil, err
		}
		for _, resource := range response.Resources {
			c.Logger.Debug("save-app-guid", lager.Data{"app-guid": resource.GUID})
			set[resource.GUID] = struct{}{}
		}
		nextPage = response.Pagination.Next.Href
		c.Logger.Debug("next-app-guid-page", lager.Data{"nextPage": nextPage})
	}

	return set, nil
}

func (c *Client) makeAppsV3Request(queryParams, token string) (AppsV3Response, error) {
	route := "/v3/apps"
	if queryParams != "" {
		route = fmt.Sprintf("%s?%s", route, queryParams)
	}
	var response AppsV3Response
	err := c.ExternalJSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return AppsV3Response{}, fmt.Errorf("json client do: %s", err)
	}
	return response, nil
}

func (c *Client) GetLiveAppGUIDs(token string, appGUIDs []string) (map[string]struct{}, error) {
	c.Logger.Info("get-live-app-guids", lager.Data{"candidate-app-guids": appGUIDs})
	token = fmt.Sprintf("bearer %s", token)

	values := url.Values{}
	values.Add("guids", strings.Join(appGUIDs, ","))
	values.Add("per_page", strconv.Itoa(len(appGUIDs)))

	route := fmt.Sprintf("/v3/apps?%s", values.Encode())
	c.Logger.Debug("live-app-guid-request", lager.Data{"route": route})

	var response AppsV3Response
	err := c.ExternalJSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}

	// TotalPages will never be greater than 1, we are setting per_page equal to size of app_guid list
	if response.Pagination.TotalPages > 1 {
		return nil, fmt.Errorf("pagination support not yet implemented")
	}

	c.Logger.Debug("live-app-guid-response", lager.Data{"resources": response.Resources})

	set := make(map[string]struct{})
	for _, r := range response.Resources {
		set[r.GUID] = struct{}{}
	}

	return set, nil
}

func (c *Client) GetLiveSpaceGUIDs(token string, spaceGUIDs []string) (map[string]struct{}, error) {
	c.Logger.Info("get-live-space-guids", lager.Data{"candidate-space-guids": spaceGUIDs})
	token = fmt.Sprintf("bearer %s", token)

	liveSpaceGUIDs := make(map[string]struct{})

	values := url.Values{}
	values.Add("guids", strings.Join(spaceGUIDs, ","))
	// Add +1 incase len is 0 - avoiding a capi error
	values.Add("per_page", strconv.Itoa(len(spaceGUIDs)+1))

	route := fmt.Sprintf("/v3/spaces?%s", values.Encode())
	c.Logger.Debug("live-space-guid-request", lager.Data{"route": route})

	var response SpacesV3Response
	err := c.ExternalJSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}

	// TotalPages will never be greater than 1, we are setting per_page equal to size of space_guids list
	if response.Pagination.TotalPages > 1 {
		return nil, fmt.Errorf("pagination support not yet implemented")
	}

	c.Logger.Debug("live-space-guid-response", lager.Data{"resources": response.Resources})

	for _, space := range response.Resources {
		liveSpaceGUIDs[space.GUID] = struct{}{}
	}

	return liveSpaceGUIDs, nil
}

func (c *Client) GetSpaceGUIDs(token string, appGUIDs []string) ([]string, error) {
	c.Logger.Info("get-space-guids", lager.Data{"app-guids": appGUIDs})
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

	c.Logger.Debug("space-guids", lager.Data{"space-guids": ret})
	return ret, nil
}

func (c *Client) GetAppSpaces(token string, appGUIDs []string) (map[string]string, error) {
	c.Logger.Info("get-app-spaces", lager.Data{"app-guids": appGUIDs})
	if len(appGUIDs) < 1 {
		return map[string]string{}, nil
	}

	token = fmt.Sprintf("bearer %s", token)

	values := url.Values{}
	values.Add("guids", strings.Join(appGUIDs, ","))
	values.Add("per_page", strconv.Itoa(len(appGUIDs)))

	route := fmt.Sprintf("/v3/apps?%s", values.Encode())
	c.Logger.Debug("get-app-spaces-request", lager.Data{"route": route})

	var response AppsV3Response
	err := c.ExternalJSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}

	// TotalPages will never be greater than 1, we are setting per_page equal to size of app_guid list
	if response.Pagination.TotalPages > 1 {
		return nil, fmt.Errorf("pagination support not yet implemented")
	}

	c.Logger.Debug("get-app-spaces-response", lager.Data{"resources": response.Resources})

	set := make(map[string]string)
	for _, r := range response.Resources {
		href := r.Links.Space.Href
		parts := strings.Split(href, "/")
		appID := r.GUID
		spaceID := parts[len(parts)-1]
		set[appID] = spaceID
	}

	c.Logger.Debug("get-app-spaces-return", lager.Data{"app-id-space-id-mapping": set})
	return set, nil
}

func (c *Client) GetSpace(token, spaceGUID string) (*SpaceResponse, error) {
	c.Logger.Info("get-space", lager.Data{"space-guid": spaceGUID})
	token = fmt.Sprintf("bearer %s", token)
	route := fmt.Sprintf("/v2/spaces/%s", spaceGUID)
	c.Logger.Debug("get-space-request", lager.Data{"route": route})

	var response SpaceResponse
	err := c.ExternalJSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		typedErr, ok := err.(*json_client.HttpResponseCodeError)
		if !ok {
			return nil, fmt.Errorf("json client do: %s", err)
		}
		if typedErr.StatusCode == http.StatusNotFound {
			c.Logger.Info("space-not-found", lager.Data{"space-guid": spaceGUID})
			return nil, nil
		}
		return nil, fmt.Errorf("json client do: %s", err)
	}
	c.Logger.Debug("get-space-response", lager.Data{"resources": response.Entity})

	return &response, nil
}

func (c *Client) GetSubjectSpace(token, subjectId string, space SpaceResponse) (*SpaceResource, error) {
	c.Logger.Info("get-subject-space", lager.Data{"subject-id": subjectId, "space-response": space})
	token = fmt.Sprintf("bearer %s", token)

	values := url.Values{}
	values.Add("q", fmt.Sprintf("developer_guid:%s", subjectId))
	values.Add("q", fmt.Sprintf("name:%s", space.Entity.Name))
	values.Add("q", fmt.Sprintf("organization_guid:%s", space.Entity.OrganizationGUID))

	route := fmt.Sprintf("/v2/spaces?%s", values.Encode())

	c.Logger.Debug("get-subject-space-request", lager.Data{"route": route})

	var response SpacesResponse
	err := c.ExternalJSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}

	c.Logger.Debug("get-subject-space-response", lager.Data{"resources": response.Resources})

	numSpaces := len(response.Resources)
	if numSpaces == 0 {
		c.Logger.Debug("get-subject-space-no-spaces")
		return nil, nil
	}
	if numSpaces > 1 {
		return nil, fmt.Errorf("found more than one matching space")
	}

	return &response.Resources[0], nil
}

func (c *Client) GetSubjectSpaces(token, subjectId string) (map[string]struct{}, error) {
	c.Logger.Info("get-subject-spaces", lager.Data{"subject-id": subjectId})
	const maximumPageSize = "100"
	token = fmt.Sprintf("bearer %s", token)

	values := url.Values{}
	values.Add("results-per-page", maximumPageSize)

	route := fmt.Sprintf("/v2/users/%s/spaces?%s", subjectId, values.Encode())

	c.Logger.Debug("get-subject-spaces-request", lager.Data{"route": route})

	var resources []SpaceResource
	for route != "" {
		var response SpacesResponse
		err := c.ExternalJSONClient.Do("GET", route, nil, &response, token)
		if err != nil {
			return nil, fmt.Errorf("json client do: %s", err)
		}

		c.Logger.Debug("get-subject-spaces-response", lager.Data{"resources": response.Resources, "next-url": response.NextUrl})

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

func (c *Client) GetSecurityGroupsWithPage(token string, page int) (GetSecurityGroupsResponse, error) {
	c.Logger.Info("get-security-groups-with-page", lager.Data{"page": page})

	token = fmt.Sprintf("bearer %s", token)
	route := "/v3/security_groups"
	queryParams := generatePageQueryParams(page)

	route = fmt.Sprintf("%s?%s", route, queryParams)
	c.Logger.Debug("get-security-groups-with-page-request", lager.Data{"route": route})

	var response GetSecurityGroupsResponse
	err := c.ExternalJSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return GetSecurityGroupsResponse{}, fmt.Errorf("json client do: %s", err)
	}

	c.Logger.Debug("get-security-groups-with-page-response", lager.Data{"resources": response.Resources})

	return response, nil
}

func (c *Client) GetSecurityGroups(token string) ([]SecurityGroupResource, error) {
	c.Logger.Info("get-security-groups")

	var err error
	var securityGroups []SecurityGroupResource

	for retry := 0; retry < 3; retry++ {
		c.Logger.Debug("get-security-groups-retry-loop", lager.Data{"attempt": retry})
		securityGroups, err = c.attemptPagination(token)
		if err == nil {
			c.Logger.Debug("get-security-groups-retry-loop-succeeded", lager.Data{"security-groups": securityGroups})
			break
		}
	}

	if err != nil {
		return []SecurityGroupResource{}, fmt.Errorf("Ran out of retry attempts. Last error was: %s\n", err.Error())
	}

	return securityGroups, nil
}

func (c *Client) attemptPagination(token string) ([]SecurityGroupResource, error) {
	c.Logger.Info("get-security-groups-attempt-pagination")
	securityGroups := []SecurityGroupResource{}

	originalUpdatedAt, err := c.GetSecurityGroupsLastUpdate(token)
	if err != nil {
		return securityGroups, err
	}

	page := 1

	for {
		// Get page
		c.Logger.Debug("get-security-groups-request", lager.Data{"page": page})
		securityGroupResponse, err := c.GetSecurityGroupsWithPage(token, page)
		if err != nil {
			return []SecurityGroupResource{}, err
		}
		c.Logger.Debug("get-security-groups-response", lager.Data{"response": securityGroupResponse.Resources})

		newUpdatedAt, err := c.GetSecurityGroupsLastUpdate(token)
		if err != nil {
			return []SecurityGroupResource{}, err
		}

		// Check to see if there are zero ASGs
		if securityGroupResponse.Resources == nil {
			c.Logger.Debug("no-additional-security-groups")
			return securityGroups, nil
		}

		// Check to make sure ASGs haven't been updated
		if newUpdatedAt.Equal(originalUpdatedAt) {
			c.Logger.Debug("get-security-groups-timestamps-match", lager.Data{"originalUpdatedAt": originalUpdatedAt, "newUpdatedAt": newUpdatedAt})
			securityGroups = append(securityGroups, securityGroupResponse.Resources...)
		} else {
			c.Logger.Debug("get-security-groups-timestamps-differ", lager.Data{"originalUpdatedAt": originalUpdatedAt, "newUpdatedAt": newUpdatedAt})
			return []SecurityGroupResource{}, NewUnstableSecurityGroupListError(errors.New("last_update time has changed"))
		}

		// Check if this is the last page
		if securityGroupResponse.Pagination.Next.Href == "" {
			c.Logger.Debug("get-security-groups-last-page", lager.Data{"page": page})
			break
		}

		page++
	}

	return securityGroups, nil
}

func (c *Client) GetSecurityGroupsLastUpdate(token string) (time.Time, error) {
	c.Logger.Info("get-security-groups-last-update")
	token = fmt.Sprintf("bearer %s", token)

	var response SecurityGroupLatestUpdateResponse
	err := c.InternalJSONClient.Do("GET", "/internal/v4/asg_latest_update", nil, &response, token)
	if err != nil {
		typedErr, ok := err.(*json_client.HttpResponseCodeError)
		if !ok {
			return time.Time{}, fmt.Errorf("json client do: %s", err)
		}

		if typedErr.StatusCode == http.StatusNotFound {
			c.Logger.Info("get-security-groups-last-update-not-found")
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("json client do: %s", err)
	}

	lastUpdateTimestamp, err := time.Parse(time.RFC3339, response.LastUpdate)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed parsing last_update from cloud controller: '%s'", response.LastUpdate)
	}

	c.Logger.Debug("get-security-groups-last-update-response", lager.Data{"last_update": lastUpdateTimestamp})

	return lastUpdateTimestamp, nil
}

func generatePageQueryParams(page int) string {
	return fmt.Sprintf("per_page=%s&page=%s", url.QueryEscape(fmt.Sprintf("%d", SECURITY_GROUPS_PER_PAGE)), url.QueryEscape(fmt.Sprintf("%d", page)))
}
