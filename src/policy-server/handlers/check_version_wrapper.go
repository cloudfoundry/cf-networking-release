package handlers

import (
	"net/http"

	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	"code.cloudfoundry.org/lager"

	"fmt"

	semver "github.com/hashicorp/go-version"
)

type CheckVersionWrapper struct {
	ErrorResponse errorResponse
}

func (c *CheckVersionWrapper) CheckVersion(handlers map[string]middleware.LoggableHandlerFunc) middleware.LoggableHandlerFunc {
	return middleware.LoggableHandlerFunc(func(logger lager.Logger, rw http.ResponseWriter, r *http.Request) {
		var version string
		switch len(r.Header["Accept"]) {
		case 0:
			version = "0.0.0"
		case 1:
			version = r.Header["Accept"][0]
		default:
			c.ErrorResponse.BadRequest(rw, nil, "check api version", "multiple accept headers not allowed")
			return
		}

		requestedVersion, err := semver.NewVersion(version)
		if err != nil {
			c.ErrorResponse.NotAcceptable(rw, nil, "check api version", fmt.Sprintf("api version '%s' not supported", version))
			return
		}

		for versionString, handler := range handlers {
			constraint, err := semver.NewConstraint(versionString)
			if err != nil {
				continue
			}
			if constraint.Check(requestedVersion) {
				handler(logger, rw, r)
				return
			}
		}

		c.ErrorResponse.NotAcceptable(rw, nil, "check api version", fmt.Sprintf("api version '%s' not supported", version))
	})
}
