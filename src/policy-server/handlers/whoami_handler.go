package handlers

import (
	"errors"
	"lib/marshal"
	"net/http"
	"strings"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/uaa_request_client.go --fake-name UAARequestClient . uaaRequestClient
type uaaRequestClient interface {
	GetName(token string) (string, error)
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
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token := authorization[0]
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")
	userName, err := h.Client.GetName(token)
	if err != nil {
		h.Logger.Error("uaa-getname", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	whoAmIResponse := WhoAmIResponse{
		UserName: userName,
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
