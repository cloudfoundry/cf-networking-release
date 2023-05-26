package json_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/lager/v3"
)

//go:generate counterfeiter -o ../fakes/http_client.go --fake-name HTTPClient . HttpClient
type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
	CloseIdleConnections()
}

//go:generate counterfeiter -o ../fakes/json_client.go --fake-name JSONClient . JsonClient
type JsonClient interface {
	Do(method, route string, reqData, respData interface{}, token string) error
	CloseIdleConnections()
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

type HttpResponseCodeError struct {
	StatusCode int
	Message    string
}

func (h *HttpResponseCodeError) Error() string {
	return fmt.Sprintf("http status %d: %s", h.StatusCode, h.Message)
}

func (c *Client) CloseIdleConnections() {
	c.HttpClient.CloseIdleConnections()
}

func (c *Client) Do(method, route string, reqData, respData interface{}, token string) error {
	var reader io.Reader
	if method != "GET" && reqData != nil {
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

	if token != "" {
		request.Header["Authorization"] = []string{token}
	}
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
		var errDescription string

		var errBody struct {
			Error string
		}
		err = json.Unmarshal(respBytes, &errBody)
		if err != nil {
			errDescription = string(respBytes)
		} else {
			errDescription = errBody.Error
		}

		c.Logger.Error("http-client", errors.New(errDescription), lager.Data{
			"body": string(respBytes),
			"code": resp.StatusCode,
		})
		return &HttpResponseCodeError{
			StatusCode: resp.StatusCode,
			Message:    errDescription,
		}
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
