package json_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"lib/marshal"
	"net/http"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/http_client.go --fake-name HTTPClient . HttpClient
type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

//go:generate counterfeiter -o ../fakes/json_client.go --fake-name JSONClient . JsonClient
type JsonClient interface {
	Do(method, route string, reqData, respData interface{}, token string) error
}

func New(logger lager.Logger, httpClient HttpClient, baseURL string) JsonClient {
	return &Client{
		Logger:      logger,
		HttpClient:  httpClient,
		Url:         baseURL,
		Marshaler:   marshal.MarshalFunc(json.Marshal),
		Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
	}
}

type Client struct {
	Logger      lager.Logger
	HttpClient  HttpClient
	Url         string
	Marshaler   marshal.Marshaler
	Unmarshaler marshal.Unmarshaler
}

func (c *Client) Do(method, route string, reqData, respData interface{}, token string) error {
	var reader io.Reader
	if method != "GET" {
		bodyBytes, err := c.Marshaler.Marshal(reqData)
		if err != nil {
			return fmt.Errorf("json marshal request body: %s", err)
		}
		reader = bytes.NewReader(bodyBytes)
	}

	reqURL := c.Url + route
	request, err := http.NewRequest(method, reqURL, reader)
	if err != nil {
		return fmt.Errorf("http new request: %s", err)
	}

	request.Header["Authorization"] = []string{token}
	resp, err := c.HttpClient.Do(request)
	if err != nil {
		return fmt.Errorf("http client do: %s", err)
	}
	defer resp.Body.Close() // untested

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("body read: %s", err)
	}

	if resp.StatusCode > 299 {
		err = fmt.Errorf("http client do: bad response status %d", resp.StatusCode)
		c.Logger.Error("http-client", err, lager.Data{
			"body": string(respBytes),
		})
		return err
	}

	c.Logger.Debug("http-do", lager.Data{
		"body": string(respBytes),
	})

	if respData != nil {
		err = c.Unmarshaler.Unmarshal(respBytes, respData)
		if err != nil {
			return fmt.Errorf("json unmarshal: %s", err)
		}
	}

	return nil
}
