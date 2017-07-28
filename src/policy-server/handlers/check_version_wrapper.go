package handlers

import (
	"net/http"

	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	"code.cloudfoundry.org/lager"

	"fmt"

	"github.com/tedsuo/rata"
)

type CheckVersionWrapper struct {
	ErrorResponse errorResponse
}

func (c *CheckVersionWrapper) CheckVersion(handlers map[string]middleware.LoggableHandlerFunc) middleware.LoggableHandlerFunc {
	return middleware.LoggableHandlerFunc(func(logger lager.Logger, rw http.ResponseWriter, r *http.Request) {
		version := rata.Param(r, "version")
		handler, ok := handlers[version]
		if ok {
			handler(logger, rw, r)
			return
		}

		c.ErrorResponse.NotAcceptable(rw, nil, "check api version", fmt.Sprintf("api version '%s' not supported", version))
	})
}
