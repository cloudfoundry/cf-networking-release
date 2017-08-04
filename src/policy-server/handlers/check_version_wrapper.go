package handlers

import (
	"net/http"

	"fmt"

	"github.com/tedsuo/rata"
)

type CheckVersionWrapper struct {
	ErrorResponse errorResponse
}

func (c *CheckVersionWrapper) CheckVersion(handlers map[string]http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		version := rata.Param(req, "version")
		handler, ok := handlers[version]
		if ok {
			handler(rw, req)
			return
		}

		c.ErrorResponse.NotAcceptable(rw, nil, "check api version", fmt.Sprintf("api version '%s' not supported", version))
	}
}
