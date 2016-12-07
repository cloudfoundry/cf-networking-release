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
	Host          string
	Name          string
	Secret        string
	HTTPClient    httpClient
	WarrantClient warrantClient
	Logger        lager.Logger
}

//go:generate counterfeiter -o ../fakes/warrant_client.go --fake-name WarrantClient . warrantClient
type warrantClient interface {
	GetToken(clientName, clientSecret string) (string, error)
}

//go:generate counterfeiter -o ../fakes/http_client.go --fake-name HTTPClient . httpClient
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type CheckTokenResponse struct {
	Scope    []string `json:"scope"`
	UserName string   `json:"user_name"`
}

func (c *Client) GetToken() (string, error) {
	token, err := c.WarrantClient.GetToken(c.Name, c.Secret)
	if err != nil {
		return "", fmt.Errorf("get token failed: %s", err)
	}
	return token, nil
}

func (c *Client) CheckToken(token string) (CheckTokenResponse, error) {
	reqURL := fmt.Sprintf("%s/check_token", c.Host)
	bodyString := "token=" + token
	request, err := http.NewRequest("POST", reqURL, strings.NewReader(bodyString))
	request.SetBasicAuth(c.Name, c.Secret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c.Logger.Debug("check-token", lager.Data{
		"URL":    request.URL,
		"Header": request.Header,
	})

	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return CheckTokenResponse{}, fmt.Errorf("http client: %s", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return CheckTokenResponse{}, fmt.Errorf("read body: %s", err)
	}

	if resp.StatusCode != 200 {
		err = BadUaaResponse{
			StatusCode:      resp.StatusCode,
			UaaResponseBody: string(respBytes),
		}
		return CheckTokenResponse{}, err
	}

	responseStruct := CheckTokenResponse{}
	err = json.Unmarshal(respBytes, &responseStruct)
	if err != nil {
		return CheckTokenResponse{}, fmt.Errorf("unmarshal json: %s", err)
	}
	return responseStruct, nil
}
