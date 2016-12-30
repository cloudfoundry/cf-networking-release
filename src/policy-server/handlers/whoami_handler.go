package handlers

import (
	"lib/marshal"
	"net/http"

	"code.cloudfoundry.org/lager"
)

type WhoAmIHandler struct {
	Logger    lager.Logger
	Marshaler marshal.Marshaler
}

type WhoAmIResponse struct {
	UserName string `json:"user_name"`
}

func (h *WhoAmIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request, currentUser string) {
	whoAmIResponse := WhoAmIResponse{
		UserName: currentUser,
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
