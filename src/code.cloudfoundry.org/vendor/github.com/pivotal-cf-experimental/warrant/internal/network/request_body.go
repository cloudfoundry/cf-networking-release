package network

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strings"
)

type requestBody interface {
	Encode() (requestBody io.Reader, contentType string, err error)
}

// JSONRequestBody is an object capable of being encoded
// as JSON within a request body.
type JSONRequestBody struct {
	body interface{}
}

// NewJSONRequestBody returns a JSONRequestBody initialized
// with an object that can be marshaled to JSON.
func NewJSONRequestBody(body interface{}) JSONRequestBody {
	return JSONRequestBody{
		body: body,
	}
}

// Encode returns an io.Reader that represents the request body and
// a string value to be used as the Content-Type header.
func (j JSONRequestBody) Encode() (requestBody io.Reader, contentType string, err error) {
	bodyJSON, err := json.Marshal(j.body)
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(bodyJSON), "application/json", nil
}

// NewFormRequestBody returns a FormRequestBody initialized with keys
// and values to be encoded.
func NewFormRequestBody(values url.Values) FormRequestBody {
	return FormRequestBody(values)
}

// FormRequestBody is an object capable of being form encoded
// into a request body.
type FormRequestBody url.Values

// Encode returns an io.Reader that represents the request body and
// a string value to be used as the Content-Type header.
func (f FormRequestBody) Encode() (requestBody io.Reader, contentType string, err error) {
	return strings.NewReader(url.Values(f).Encode()), "application/x-www-form-urlencoded", nil
}
