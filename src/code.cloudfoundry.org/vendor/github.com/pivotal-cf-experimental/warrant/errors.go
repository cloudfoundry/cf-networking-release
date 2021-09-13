package warrant

import (
	"fmt"
	"net/http"

	"github.com/pivotal-cf-experimental/warrant/internal/network"
)

// UnexpectedStatusError indicates that UAA returned a status code that was unexpected.
// The error message should provide some information about the specific error.
type UnexpectedStatusError struct {
	err error
}

// Error returns a string representation of the UnexpectedStatusError.
func (e UnexpectedStatusError) Error() string {
	return e.err.Error()
}

// UnauthorizedError indicates that the requested action was unauthorized.
// This could mean that the provided token is invalid, or does not contain
// the required scope.
type UnauthorizedError struct {
	err error
}

// Error returns a string representation of the UnauthorizedError.
func (e UnauthorizedError) Error() string {
	return e.err.Error()
}

// NotFoundError indicates that the resource could not be found.
type NotFoundError struct {
	err error
}

// Error returns a string representation of the NotFoundError.
func (e NotFoundError) Error() string {
	return e.err.Error()
}

// UnknownError indicates that an error of unknown type has been encountered.
type UnknownError struct {
	err error
}

// Error returns a string representation of the UnknownError.
func (e UnknownError) Error() string {
	return e.err.Error()
}

// InvalidTokenError indicates that the provided token is invalid.
// The specific issue can be found by viewing the Error() return value.
type InvalidTokenError struct {
	err error
}

// Error returns a string representation of the InvalidTokenError.
func (e InvalidTokenError) Error() string {
	return e.err.Error()
}

// MalformedResponseError indicates that the response received from UAA is malformed.
type MalformedResponseError struct {
	err error
}

// Error returns a string representation of the MalformedResponseError.
func (e MalformedResponseError) Error() string {
	return fmt.Sprintf("malformed response: %s", e.err)
}

// BadRequestError indicates that the request sent to UAA is invalid.
// The specific issue can be found by inspecting the Error() output.
type BadRequestError struct {
	err error
}

// Error returns a string representation of the BadRequestError.
func (e BadRequestError) Error() string {
	return fmt.Sprintf("bad request: %s", e.err.(network.UnexpectedStatusError).Body)
}

// DuplicateResourceError indicates that the action committed against the resource
// would result in a duplicate.
type DuplicateResourceError struct {
	err error
}

// Error returns a string representation of the DuplicateResourceError.
func (e DuplicateResourceError) Error() string {
	return fmt.Sprintf("duplicate resource: %s", e.err.(network.UnexpectedStatusError).Body)
}

func translateError(err error) error {
	switch s := err.(type) {
	case network.NotFoundError:
		return NotFoundError{err}
	case network.UnauthorizedError:
		return UnauthorizedError{err}
	case network.UnexpectedStatusError:
		switch s.Status {
		case http.StatusBadRequest:
			return BadRequestError{err}
		case http.StatusConflict:
			return DuplicateResourceError{err}
		default:
			return UnexpectedStatusError{err}
		}
	default:
		return UnknownError{err}
	}
}
