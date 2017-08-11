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
	UserName string `json:"user_name"`
}

func (h *WhoAmIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("who-am-i")
	tokenData := getTokenData(req)
	whoAmIResponse := WhoAmIResponse{
		UserName: tokenData.UserName,
	}
	responseJSON, err := h.Marshaler.Marshal(whoAmIResponse)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "marshaling response failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}
