package cc_client

import (
	"fmt"
	"lib/json_client"
	"net/http"
	"net/url"
	"policy-server/models"
	"strconv"
	"strings"

	"code.cloudfoundry.org/lager"
)

type Client struct {
	Logger     lager.Logger
	JSONClient json_client.JsonClient
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

type SpaceResponse struct {
	Entity struct {
		Name             string `json:"name"`
		OrganizationGUID string `json:"organization_guid"`
	} `json:"entity"`
}

type SpacesResponse struct {
	Resources []struct {
		Metadata struct {
			GUID string `json:"guid"`
		}
		Entity struct {
			Name             string `json:"name"`
			OrganizationGUID string `json:"organization_guid"`
		} `json:"entity"`
	} `json:"resources"`
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

	if response.Pagination.TotalPages > 1 {
		return nil, fmt.Errorf("pagination support not yet implemented")
	}

	set := make(map[string]struct{})
	for _, r := range response.Resources {
		set[r.GUID] = struct{}{}
	}

	return set, nil
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

	route := fmt.Sprintf("/v3/apps?%s", values.Encode())

	var response AppsV3Response
	err := c.JSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}

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

func (c *Client) GetSpace(token, spaceGUID string) (*models.Space, error) {
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

	return &models.Space{
		Name:    response.Entity.Name,
		OrgGUID: response.Entity.OrganizationGUID,
	}, nil
}

func (c *Client) GetUserSpace(token, userGUID string, space models.Space) (*models.Space, error) {
	token = fmt.Sprintf("bearer %s", token)

	values := url.Values{}
	values.Add("q", fmt.Sprintf("developer_guid:%s", userGUID))
	values.Add("q", fmt.Sprintf("name:%s", space.Name))
	values.Add("q", fmt.Sprintf("organization_guid:%s", space.OrgGUID))

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

	return &models.Space{
		Name:    response.Resources[0].Entity.Name,
		OrgGUID: response.Resources[0].Entity.OrganizationGUID,
	}, nil
}

func (c *Client) GetUserSpaces(token, userGUID string) (map[string]struct{}, error) {
	token = fmt.Sprintf("bearer %s", token)

	route := fmt.Sprintf("/v2/users/%s/spaces", userGUID)

	var response SpacesResponse
	err := c.JSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, fmt.Errorf("json client do: %s", err)
	}

	userSpaces := map[string]struct{}{}
	for _, space := range response.Resources {
		spaceID := space.Metadata.GUID
		userSpaces[spaceID] = struct{}{}
	}

	return userSpaces, nil
}
