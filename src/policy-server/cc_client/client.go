package cc_client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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

func (r BadCCResponse) Error() string {
	return fmt.Sprintf("bad cc response: %d: %s", r.StatusCode, r.CCResponseBody)
}

func (c *Client) GetAllAppGUIDs(token string) (map[string]interface{}, error) {

	reqURL := fmt.Sprintf("%s/v3/apps", c.Host)
	request, err := http.NewRequest("GET", reqURL, nil)
	request.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %s", err)
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

	response := struct {
		Pagination struct {
			TotalPages int `json:"total_pages"`
		} `json:"pagination"`
		Resources []struct {
			GUID string `json:"guid"`
		} `json:"resources"`
	}{}

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
