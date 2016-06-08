package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

//go:generate counterfeiter -o ../fakes/uaa_request_client.go --fake-name UAARequestClient . uaaRequestClient
type uaaRequestClient interface {
	GetName(token string) (string, error)
}

type WhoAmIHandler struct {
	Client uaaRequestClient
}

type WhoAmIResponse struct {
	UserName string `json:"user_name"`
}

func (h *WhoAmIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	authorization := req.Header["Authorization"]
	if len(authorization) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token := authorization[0]
	token = strings.TrimPrefix(token, "Bearer ")
	userName, err := h.Client.GetName(token)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	whoAmIResponse := WhoAmIResponse{
		UserName: userName,
	}
	responseJSON, err := json.Marshal(whoAmIResponse)
	if err != nil {
		//not tested
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
	return
}
