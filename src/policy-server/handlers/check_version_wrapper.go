package handlers

import (
	"net/http"

	"fmt"
)

//go:generate counterfeiter -o fakes/rata_adapter.go --fake-name RataAdapter . rataAdapter
type rataAdapter interface {
	Param(*http.Request, string) string
}

type CheckVersionWrapper struct {
	ErrorResponse errorResponse
	RataAdapter   rataAdapter
}

func (c *CheckVersionWrapper) CheckVersion(handlers map[string]http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		version := c.RataAdapter.Param(req, "version")
		handler, ok := handlers[version]
		if ok {
			handler.ServeHTTP(rw, req)
			return
		}

		logger := getLogger(req)
		logger = logger.Session("check-version")
		c.ErrorResponse.NotAcceptable(logger, rw, nil, fmt.Sprintf("api version '%s' not supported", version))
	})
}
