package handlers

import (
	"lib/marshal"
	"net/http"
	"policy-server/server_metrics"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager"
)

type WhoAmIHandler struct {
	Logger        lager.Logger
	Marshaler     marshal.Marshaler
	MetricsSender metricsSender
}

type WhoAmIResponse struct {
	UserName string `json:"user_name"`
}

func (h *WhoAmIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
	whoAmIResponse := WhoAmIResponse{
		UserName: tokenData.UserName,
	}
	responseJSON, err := h.Marshaler.Marshal(whoAmIResponse)
	if err != nil {
		h.Logger.Error("marshal-response", err)
		w.WriteHeader(http.StatusInternalServerError)
		h.MetricsSender.IncrementCounter(server_metrics.MetricExternalWhoAmIError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
	return
}
