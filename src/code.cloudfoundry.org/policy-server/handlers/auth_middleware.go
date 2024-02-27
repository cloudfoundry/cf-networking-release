package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagerflags"
	"code.cloudfoundry.org/lib/common"
	"code.cloudfoundry.org/policy-server/uaa_client"
)

type Key string

const TokenDataKey = Key("tokenData")

const MAX_REQ_BODY_SIZE = 10 << 20 // 10 MB

//counterfeiter:generate -o fakes/http_handler.go --fake-name HTTPHandler . http_handler
type http_handler interface {
	http.Handler
}

type UAAClient interface {
	CheckToken(token string) (uaa_client.CheckTokenResponse, error)
}

type Authenticator struct {
	Client        UAAClient
	Scopes        []string
	ErrorResponse errorResponse
	ScopeChecking bool
}

func getLogger(req *http.Request) lager.Logger {
	if v := req.Context().Value(middleware.Key("logger")); v != nil {
		if logger, ok := v.(lager.Logger); ok {
			return logger
		}
	}
	logger, _ := lagerflags.NewFromConfig("cfnetworking.policy-server", common.GetLagerConfig())
	return logger
}

func getTokenData(req *http.Request) uaa_client.CheckTokenResponse {
	if v := req.Context().Value(TokenDataKey); v != nil {
		if token, ok := v.(uaa_client.CheckTokenResponse); ok {
			return token
		}
	}
	return uaa_client.CheckTokenResponse{}
}

func (a *Authenticator) Wrap(handle http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		logger := getLogger(req)
		logger = logger.Session("authentication")

		authorization := req.Header["Authorization"]
		if len(authorization) < 1 {
			err := errors.New("no auth header")
			a.ErrorResponse.Unauthorized(logger, w, err, "missing authorization header")
			return
		}

		token := authorization[0]
		token = strings.TrimPrefix(token, "Bearer ")
		token = strings.TrimPrefix(token, "bearer ")
		tokenData, err := a.Client.CheckToken(token)
		if err != nil {
			a.ErrorResponse.Unauthorized(logger, w, err, "failed to verify token with uaa")
			return
		}

		if a.ScopeChecking && !isAuthorized(tokenData.Scope, a.Scopes) {
			err := fmt.Errorf("provided scopes %s do not include allowed scopes %s", tokenData.Scope, a.Scopes)
			a.ErrorResponse.Forbidden(logger, w, err, err.Error())
			return
		}

		req.Body = http.MaxBytesReader(w, req.Body, MAX_REQ_BODY_SIZE)

		contextWithTokenData := context.WithValue(req.Context(), TokenDataKey, tokenData)
		req = req.WithContext(contextWithTokenData)
		handle.ServeHTTP(w, req)
	})
}

func isAuthorized(scopes, allowedScopes []string) bool {
	for _, scope := range scopes {
		for _, allowed := range allowedScopes {
			if scope == allowed {
				return true
			}
		}
	}
	return false
}

func isNetworkAdmin(scopes []string) bool {
	for _, scope := range scopes {
		if scope == "network.admin" {
			return true
		}
	}
	return false
}

func isNetworkWrite(scopes []string) bool {
	for _, scope := range scopes {
		if scope == "network.write" {
			return true
		}
	}
	return false
}
