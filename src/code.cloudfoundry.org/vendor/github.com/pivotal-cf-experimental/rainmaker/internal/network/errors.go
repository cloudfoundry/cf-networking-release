package network

import "fmt"

// RequestBodyEncodeError indicates that the body passed in
// the Request cannot be encoded.
type RequestBodyEncodeError struct {
	err error
}

func newRequestBodyEncodeError(err error) RequestBodyEncodeError {
	return RequestBodyEncodeError{err: err}
}

// Error returns a string representation of the RequestBodyEncodeError.
func (e RequestBodyEncodeError) Error() string {
	return fmt.Sprintf("error marshalling request body: %v", e.err)
}

// RequestConfigurationError indicates that an HTTP request
// cannot be created.
type RequestConfigurationError struct {
	err error
}

func newRequestConfigurationError(err error) RequestConfigurationError {
	return RequestConfigurationError{err: err}
}

// Error returns a string representation of the RequestConfigurationError.
func (e RequestConfigurationError) Error() string {
	return fmt.Sprintf("invalid request configuration: %v", e.err)
}

// RequestHTTPError indicates that some portion of the
// HTTP request to the remote has failed.
type RequestHTTPError struct {
	err error
}

func newRequestHTTPError(err error) RequestHTTPError {
	return RequestHTTPError{err: err}
}

// Error returns a string representation of the RequestHTTPError.
func (e RequestHTTPError) Error() string {
	return fmt.Sprintf("error during http request: %v", e.err)
}

// ResponseReadError indicates that the response body could not be read.
type ResponseReadError struct {
	err error
}

func newResponseReadError(err error) ResponseReadError {
	return ResponseReadError{err: err}
}

// Error returns a string representation of the ResponseReadError.
func (e ResponseReadError) Error() string {
	return fmt.Sprintf("error reading http response: %v", e.err)
}

// UnexpectedStatusError indicates that the response status code
// that was returned from the remote host was not in the list of
// AcceptableStatusCodes specified in the Request.
type UnexpectedStatusError struct {
	Status int
	Body   []byte
}

func newUnexpectedStatusError(status int, body []byte) UnexpectedStatusError {
	return UnexpectedStatusError{
		Status: status,
		Body:   body,
	}
}

// Error returns a string representation of the UnexpectedStatusError.
func (e UnexpectedStatusError) Error() string {
	return fmt.Sprintf("unexpected status: %d %s", e.Status, e.Body)
}

// NotFoundError indicates that the requested API endpoint or resource
// could not be found.
type NotFoundError struct {
	message []byte
}

func newNotFoundError(message []byte) NotFoundError {
	return NotFoundError{message: message}
}

// Error returns a string representation of the NotFoundError.
func (e NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s", e.message)
}

// UnauthorizedError indicates that the request could not be
// completed because the authorization that was provided does
// not meet the expected permissions requirements from UAA.
type UnauthorizedError struct {
	message []byte
}

func newUnauthorizedError(message []byte) UnauthorizedError {
	return UnauthorizedError{message: message}
}

// Error returns a string representation of the UnauthorizedError.
func (e UnauthorizedError) Error() string {
	return fmt.Sprintf("unauthorized: %s", e.message)
}
