package cc_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"lib/models"
	"net/http"
	"strings"

	"code.cloudfoundry.org/lager"
)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	Host       string
	HTTPClient httpClient
	Logger     lager.Logger
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
		OrganizationGuid string `json:"organization_guid"`
	} `json:"entity"`
}

func (r BadCCResponse) Error() string {
	return fmt.Sprintf("bad cc response: %d: %s", r.StatusCode, r.CCResponseBody)
}

func (c *Client) GetAllAppGUIDs(token string) (map[string]interface{}, error) {
	reqURL := fmt.Sprintf("%s/v3/apps", c.Host)
	request, err := http.NewRequest("GET", reqURL, nil)
	request.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %s", err) // untested
	}

	c.Logger.Debug("get_cc_apps", lager.Data{"URL": request.URL})

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

	var response AppsV3Response
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json: %s", err)
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
	if len(appGUIDs) < 1 {
		return nil, errors.New("list of app GUIDs must not be empty")
	}
	reqURL := fmt.Sprintf("%s/v3/apps?guids=%s", c.Host, strings.Join(appGUIDs, ","))
	request, err := http.NewRequest("GET", reqURL, nil)
	request.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %s", err) // untested
	}

	c.Logger.Debug("get_cc_apps_with_guids", lager.Data{"URL": request.URL})

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

	var response AppsV3Response
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json: %s", err)
	}

	if response.Pagination.TotalPages > 1 {
		return nil, fmt.Errorf("pagination support not yet implemented")
	}

	set := make(map[string]struct{})
	for _, r := range response.Resources {
		href := r.Links.Space.Href
		parts := strings.Split(href, "/")
		set[parts[len(parts)-1]] = struct{}{}
	}
	ret := make([]string, 0, len(set))
	for guid, _ := range set {
		ret = append(ret, guid)
	}

	return ret, nil
}

func (c *Client) GetSpace(token, spaceGuid string) (models.Space, error) {
	none := models.Space{}
	reqURL := fmt.Sprintf("%s/v2/spaces/%s", c.Host, spaceGuid)
	request, err := http.NewRequest("GET", reqURL, nil)
	request.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))
	if err != nil {
		return none, fmt.Errorf("create HTTP request: %s", err) // untested
	}

	c.Logger.Debug("get_space", lager.Data{"URL": request.URL})

	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return none, fmt.Errorf("http client: %s", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return none, fmt.Errorf("read body: %s", err)
	}

	if resp.StatusCode != 200 {
		err = BadCCResponse{
			StatusCode:     resp.StatusCode,
			CCResponseBody: string(respBytes),
		}
		return none, err
	}

	var response SpaceResponse
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return none, fmt.Errorf("unmarshal json: %s", err)
	}

	return models.Space{
		Name:    response.Entity.Name,
		OrgGuid: response.Entity.OrganizationGuid,
	}, nil
}
