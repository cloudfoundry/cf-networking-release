package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"policy-server/uaa_client"
	"strings"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/http_handler.go --fake-name HTTPHandler . http_handler
type http_handler interface {
	http.Handler
}

type UAAClient interface {
	CheckToken(token string) (uaa_client.CheckTokenResponse, error)
}

type Authenticator struct {
	Client UAAClient
	Logger lager.Logger
	Scopes []string
}

//go:generate counterfeiter -o ../fakes/authenticated_handler.go --fake-name AuthenticatedHandler . authenticatedHandler
type authenticatedHandler interface {
	ServeHTTP(response http.ResponseWriter, request *http.Request, tokenData uaa_client.CheckTokenResponse)
}

func (a *Authenticator) Wrap(handle authenticatedHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		a.Logger.Debug("request made to policy-server", lager.Data{"URL": req.URL, "RemoteAddr": req.RemoteAddr})

		authorization := req.Header["Authorization"]
		if len(authorization) < 1 {
			a.Logger.Error("auth", errors.New("no auth header"))
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{ "error": "missing authorization header" }`))
			return
		}

		token := authorization[0]
		token = strings.TrimPrefix(token, "Bearer ")
		token = strings.TrimPrefix(token, "bearer ")
		tokenData, err := a.Client.CheckToken(token)
		if err != nil {
			a.Logger.Error("uaa-getname", err)
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{ "error": "failed to verify token with uaa" }`))
			return
		}

		a.Logger.Debug("request made with token:", lager.Data{"tokenData": tokenData})
		if !isAuthorized(tokenData.Scope, a.Scopes) {
			a.Logger.Error("authorization", errors.New("no allowed scopes found"),
				lager.Data{
					"allowed-scopes":  a.Scopes,
					"provided-scopes": tokenData.Scope,
				})
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(fmt.Sprintf(`{ "error": "token missing allowed scopes: %s" }`, a.Scopes)))
			return
		}

		handle.ServeHTTP(w, req, tokenData)
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
