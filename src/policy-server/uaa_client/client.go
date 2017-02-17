package uaa_client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"code.cloudfoundry.org/lager"
)

type BadUaaResponse struct {
	StatusCode      int
	UaaResponseBody string
}

func (r BadUaaResponse) Error() string {
	return fmt.Sprintf("bad uaa response: %d: %s", r.StatusCode, r.UaaResponseBody)
}

type Client struct {
	BaseURL    string
	Name       string
	Secret     string
	HTTPClient httpClient
	Logger     lager.Logger
}

//go:generate counterfeiter -o fakes/http_client.go --fake-name HTTPClient . httpClient
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type CheckTokenResponse struct {
	Scope    []string `json:"scope"`
	UserID   string   `json:"user_id"`
	UserName string   `json:"user_name"`
}

func (c *Client) GetToken() (string, error) {
	reqURL := fmt.Sprintf("%s/oauth/token", c.BaseURL)
	bodyString := fmt.Sprintf("client_id=%s&grant_type=client_credentials", c.Name)
	request, err := http.NewRequest("POST", reqURL, strings.NewReader(bodyString))
	request.SetBasicAuth(c.Name, c.Secret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c.Logger.Debug("get-token", lager.Data{"URL": request.URL})

	type getTokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	response := &getTokenResponse{}
	err = c.makeRequest(request, response)
	if err != nil {
		return "", err
	}
	return response.AccessToken, nil
}

func (c *Client) CheckToken(token string) (CheckTokenResponse, error) {
	reqURL := fmt.Sprintf("%s/check_token", c.BaseURL)
	bodyString := "token=" + token
	request, err := http.NewRequest("POST", reqURL, strings.NewReader(bodyString))
	request.SetBasicAuth(c.Name, c.Secret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c.Logger.Debug("check-token", lager.Data{"URL": request.URL})

	response := &CheckTokenResponse{}
	err = c.makeRequest(request, response)
	if err != nil {
		return CheckTokenResponse{}, err
	}
	return *response, nil
}

func (c *Client) makeRequest(request *http.Request, response interface{}) error {
	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("http client: %s", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %s", err)
	}

	if resp.StatusCode != 200 {
		err = BadUaaResponse{
			StatusCode:      resp.StatusCode,
			UaaResponseBody: string(respBytes),
		}
		return err
	}

	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return fmt.Errorf("unmarshal json: %s", err)
	}
	return nil
}
