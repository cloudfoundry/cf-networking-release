package handlers

import (
	"errors"
	"net/http"
	"policy-server/uaa_client"
	"strings"

	"github.com/pivotal-golang/lager"
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
}

func (a *Authenticator) Wrap(handle http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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

		if !isAuthorized(tokenData) {
			a.Logger.Error("authorization", errors.New("network.admin scope not found"),
				lager.Data{
					"provided-scopes": tokenData.Scope,
				})
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{ "error": "token missing required scope network.admin" }`))
			return
		}

		handle.ServeHTTP(w, req)
	})
}
