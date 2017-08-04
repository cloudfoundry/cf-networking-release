package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"policy-server/uaa_client"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/middleware"
	"code.cloudfoundry.org/lager"
)

type Key string

const TokenDataKey = Key("tokenData")

const MAX_REQ_BODY_SIZE = 10 << 20 // 10 MB

//go:generate counterfeiter -o fakes/http_handler.go --fake-name HTTPHandler . http_handler
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
	return lager.NewLogger("cfnetworking.policy-server")
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
			logger.Error("failed-missing-authorization-header", err)
			a.ErrorResponse.Unauthorized(w, err, "authenticator", "missing authorization header")
			return
		}

		token := authorization[0]
		token = strings.TrimPrefix(token, "Bearer ")
		token = strings.TrimPrefix(token, "bearer ")
		tokenData, err := a.Client.CheckToken(token)
		if err != nil {
			logger.Error("failed-verifying-token-with-uaa", err)
			a.ErrorResponse.Forbidden(w, err, "authenticator", "failed to verify token with uaa")
			return
		}

		if a.ScopeChecking && !isAuthorized(tokenData.Scope, a.Scopes) {
			err := errors.New(fmt.Sprintf("provided scopes %s do not include allowed scopes %s", tokenData.Scope, a.Scopes))
			logger.Error("failed-authorizing-provided-scope", err)
			a.ErrorResponse.Forbidden(w, err, "authenticator", err.Error())
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
