package cc_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"lib/json_client"
	"net/http"
	"policy-server/models"
	"strings"

	"code.cloudfoundry.org/lager"
)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	BaseURL    string
	HTTPClient httpClient
	Logger     lager.Logger
	JSONClient json_client.JsonClient
}

type BadCCResponse struct {
	StatusCode     int
	CCResponseBody string
}

type AppsV3Response struct {
	Pagination struct {
		TotalPages int `json:"total_pages"`
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

func (r BadCCResponse) Error() string {
	return fmt.Sprintf("bad cc response: %d: %s", r.StatusCode, r.CCResponseBody)
}

func (c *Client) GetAllAppGUIDs(token string) (map[string]interface{}, error) {
	token = fmt.Sprintf("bearer %s", token)

	var response AppsV3Response
	err := c.JSONClient.Do("GET", "/v3/apps", nil, &response, token)
	if err != nil {
		return nil, err
	}

	if response.Pagination.TotalPages > 1 {
		return nil, fmt.Errorf("pagination support not yet implemented")
	}

	ret := make(map[string]interface{})
	for _, r := range response.Resources {
		ret[r.GUID] = nil
	}

	return ret, nil
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
	route := fmt.Sprintf("/v3/apps?guids=%s", strings.Join(appGUIDs, ","))

	var response AppsV3Response
	err := c.JSONClient.Do("GET", route, nil, &response, token)
	if err != nil {
		return nil, err
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
	reqURL := fmt.Sprintf("%s/v2/spaces/%s", c.BaseURL, spaceGUID)
	request, err := http.NewRequest("GET", reqURL, nil)
	request.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %s", err) // untested
	}

	c.Logger.Debug("get_space", lager.Data{"URL": request.URL})

	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("http client: %s", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %s", err)
	}

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode != 200 {
		err = BadCCResponse{
			StatusCode:     resp.StatusCode,
			CCResponseBody: string(respBytes),
		}
		return nil, err
	}

	var response SpaceResponse
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json: %s", err)
	}

	return &models.Space{
		Name:    response.Entity.Name,
		OrgGUID: response.Entity.OrganizationGUID,
	}, nil
}

func (c *Client) GetUserSpace(token, userGUID string, space models.Space) (*models.Space, error) {
	reqURL := fmt.Sprintf("%s/v2/spaces?q=developer_guid%%3A%s&q=name%%3A%s&q=organization_guid%%3A%s", c.BaseURL, userGUID, space.Name, space.OrgGUID)
	request, err := http.NewRequest("GET", reqURL, nil)
	request.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %s", err) // untested
	}

	c.Logger.Debug("get_user_space_with_name_and_org_guid", lager.Data{"URL": request.URL})

	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("http client: %s", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %s", err)
	}

	if resp.StatusCode != 200 {
		err = BadCCResponse{
			StatusCode:     resp.StatusCode,
			CCResponseBody: string(respBytes),
		}
		return nil, err
	}

	var response SpacesResponse
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json: %s", err)
	}

	numSpaces := len(response.Resources)
	if numSpaces == 0 {
		return nil, nil
	}
	if numSpaces > 1 {
		return nil, errors.New(fmt.Sprintf("found more than one matching space"))
	}

	return &models.Space{
		Name:    response.Resources[0].Entity.Name,
		OrgGUID: response.Resources[0].Entity.OrganizationGUID,
	}, nil
}

func (c *Client) GetUserSpaces(token, userGUID string) (map[string]struct{}, error) {
	reqURL := fmt.Sprintf("%s/v2/users/%s/spaces", c.BaseURL, userGUID)
	request, err := http.NewRequest("GET", reqURL, nil)
	request.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %s", err) // untested
	}

	c.Logger.Debug("get_user_spaces", lager.Data{"URL": request.URL})

	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("http client: %s", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %s", err)
	}

	if resp.StatusCode != 200 {
		err = BadCCResponse{
			StatusCode:     resp.StatusCode,
			CCResponseBody: string(respBytes),
		}
		return nil, err
	}

	var response SpacesResponse
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json: %s", err)
	}

	userSpaces := map[string]struct{}{}
	for _, space := range response.Resources {
		spaceID := space.Metadata.GUID
		userSpaces[spaceID] = struct{}{}
	}

	return userSpaces, nil
}
