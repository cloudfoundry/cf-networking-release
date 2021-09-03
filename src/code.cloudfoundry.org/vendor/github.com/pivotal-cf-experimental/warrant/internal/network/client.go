package network

import (
	"io"
	"io/ioutil"
	"net/http"
)

// Client provides the ability to make HTTP requests.
type Client struct {
	config Config
}

// Config sets the configuration for a Client.
type Config struct {
	// Host is the fully qualified host of the remote server.
	Host string

	// SkipVerifySSL is a boolean value indicating whether SSL certificates
	// will be validated when requests are made to servers secured by HTTPS.
	SkipVerifySSL bool

	// TraceWriter is an io.Writer to which trace information can be written.
	// This is an optional field.
	TraceWriter io.Writer
}

// Request describes the requested operation to commit against the remote
// server.
type Request struct {
	// Method is an HTTP method like GET, POST, PUT, DELETE, HEAD, or OPTIONS.
	Method string

	// Path is the path portion of the URL to request against the remote host
	// including any query parameters. This field is represented as a URL
	// encoded string.
	Path string

	// Authorization provides a method for authenticating requests to UAA.
	// Supported authorization types include Basic and Bearer token authorization.
	// New types of authorization can be implemented by conforming to the following
	// interface:
	//	Authorization() string
	Authorization authorization

	// IfMatch provides access to the "If-Match" header of a request. This
	// header is used to implement a conditional-update semantic for modifying
	// UAA resources.
	IfMatch string

	// Body is a JSON or Form encoded representation of some request payload.
	// New types of request body can be implementated by conforming to the
	// following interface:
	//	Encode() (body io.Reader, contentType string, err error)
	Body requestBody

	// AcceptableStatusCodes is a list of the status codes expected to be received
	// from the remote host. Response status codes that are not included in this
	// list will cause an UnexpectedStatusError. Additionally, this is a required
	// field, and failure to populate this list will result in a panic upon execution.
	AcceptableStatusCodes []int

	// DoNotFollowRedirects is a boolean value to indicate to the client whether 3xx
	// response codes should be followed, or treated as terminal responses. The client
	// will make a single roundtrip in the case that this value is set to true.
	DoNotFollowRedirects bool
}

// Response describes the response information provided by the remote host.
type Response struct {
	// Code is the HTTP status of the response.
	Code int

	// Body is the entire contents of the response body.
	Body []byte

	// Headers is a key/value store of the headers returned in the response.
	Headers http.Header
}

// NewClient returns a Client initialized with the given Config.
func NewClient(config Config) Client {
	return Client{
		config: config,
	}
}

// MakeRequest initiates a request to the remote host, returning a response and
// possible error.
func (c Client) MakeRequest(req Request) (Response, error) {
	if req.AcceptableStatusCodes == nil {
		panic("acceptable status codes for this request were not set")
	}

	request, err := c.buildRequest(req)
	if err != nil {
		return Response{}, err
	}

	var resp *http.Response
	transport := buildTransport(c.config.SkipVerifySSL)
	if req.DoNotFollowRedirects {
		resp, err = transport.RoundTrip(request)
	} else {
		client := &http.Client{Transport: transport}
		resp, err = client.Do(request)
	}
	if err != nil {
		return Response{}, newRequestHTTPError(err)
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Response{}, newResponseReadError(err)
	}

	return c.handleResponse(req, Response{
		Code:    resp.StatusCode,
		Body:    responseBody,
		Headers: resp.Header,
	})
}

func (c Client) buildRequest(req Request) (*http.Request, error) {
	var bodyReader io.Reader
	var contentType string
	if req.Body != nil {
		var err error
		bodyReader, contentType, err = req.Body.Encode()
		if err != nil {
			return &http.Request{}, newRequestBodyEncodeError(err)
		}
	}

	requestURL := c.config.Host + req.Path
	request, err := http.NewRequest(req.Method, requestURL, bodyReader)
	if err != nil {
		return &http.Request{}, newRequestConfigurationError(err)
	}

	if req.Authorization != nil {
		request.Header.Set("Authorization", req.Authorization.Authorization())
	}

	request.Header.Set("Accept", "application/json")

	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if req.IfMatch != "" {
		request.Header.Set("If-Match", req.IfMatch)
	}

	c.printRequest(request)

	return request, nil
}

func (c Client) handleResponse(request Request, response Response) (Response, error) {
	c.printResponse(response)

	if response.Code == http.StatusNotFound {
		return Response{}, newNotFoundError(response.Body)
	}

	if response.Code == http.StatusUnauthorized || response.Code == http.StatusForbidden {
		return Response{}, newUnauthorizedError(response.Body)
	}

	for _, acceptableCode := range request.AcceptableStatusCodes {
		if response.Code == acceptableCode {
			return response, nil
		}
	}

	return Response{}, newUnexpectedStatusError(response.Code, response.Body)
}
