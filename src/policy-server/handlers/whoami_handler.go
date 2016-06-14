package handlers

import (
	"errors"
	"lib/marshal"
	"net/http"
	"policy-server/uaa_client"
	"strings"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/uaa_request_client.go --fake-name UAARequestClient . uaaRequestClient
type uaaRequestClient interface {
	CheckToken(token string) (uaa_client.CheckTokenResponse, error)
}

type WhoAmIHandler struct {
	Client    uaaRequestClient
	Logger    lager.Logger
	Marshaler marshal.Marshaler
}

type WhoAmIResponse struct {
	UserName string `json:"user_name"`
}

func (h *WhoAmIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	authorization := req.Header["Authorization"]
	if len(authorization) < 1 {
		h.Logger.Error("auth", errors.New("no auth header"))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token := authorization[0]
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")
	tokenData, err := h.Client.CheckToken(token)
	if err != nil {
		h.Logger.Error("uaa-getname", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !isAuthorized(tokenData) {
		h.Logger.Error("authorization", errors.New("network.admin scope not found"),
			lager.Data{
				"provided-scopes": tokenData.Scope,
			})
		w.WriteHeader(http.StatusForbidden)
		return
	}

	whoAmIResponse := WhoAmIResponse{
		UserName: tokenData.UserName,
	}
	responseJSON, err := h.Marshaler.Marshal(whoAmIResponse)
	if err != nil {
		h.Logger.Error("marshal-response", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
	return
}

func isAuthorized(tokenData uaa_client.CheckTokenResponse) bool {
	for _, scope := range tokenData.Scope {
		if scope == "network.admin" {
			return true
		}
	}
	return false
}
