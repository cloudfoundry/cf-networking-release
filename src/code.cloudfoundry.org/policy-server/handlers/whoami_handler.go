package handlers

import (
	"net/http"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
)

type WhoAmIHandler struct {
	Marshaler     marshal.Marshaler
	ErrorResponse errorResponse
}

type WhoAmIResponse struct {
	Subject string `json:"subject"`
}

func (h *WhoAmIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("who-am-i")
	tokenData := getTokenData(req)
	whoAmIResponse := WhoAmIResponse{
		Subject: tokenData.UserName,
	}
	if len(whoAmIResponse.Subject) < 1 {
		whoAmIResponse.Subject = tokenData.Subject
	}
	responseJSON, err := h.Marshaler.Marshal(whoAmIResponse)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "marshaling response failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	// #nosec G104 - ignore errors writing http responses to avoid spamming logs during a DoS
	w.Write(responseJSON)
}
