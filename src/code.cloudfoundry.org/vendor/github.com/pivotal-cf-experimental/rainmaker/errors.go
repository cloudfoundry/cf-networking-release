package rainmaker

import "github.com/pivotal-cf-experimental/rainmaker/internal/network"

type NotFoundError struct {
	Err error
}

func (e NotFoundError) Error() string {
	return e.Err.Error()
}

type Error struct {
	Err error
}

func (e Error) Error() string {
	return e.Err.Error()
}

func translateError(err error) error {
	switch err.(type) {
	case network.NotFoundError:
		return NotFoundError{err}
	default:
		return Error{err}
	}
}
