package handlers

import (
	"net/http"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/lager"
)

type WhoAmIHandler struct {
	Marshaler     marshal.Marshaler
	ErrorResponse errorResponse
}

type WhoAmIResponse struct {
	UserName string `json:"user_name"`
}

func (h *WhoAmIHandler) ServeHTTP(logger lager.Logger, w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
	logger = logger.Session("who-am-i")
	whoAmIResponse := WhoAmIResponse{
		UserName: tokenData.UserName,
	}
	responseJSON, err := h.Marshaler.Marshal(whoAmIResponse)
	if err != nil {
		logger.Error("failed-marshalling-response", err)
		h.ErrorResponse.InternalServerError(w, err, "who-am-i", "marshaling response failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}
