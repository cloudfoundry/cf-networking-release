package handlers

import (
	"encoding/json"
	"example-apps/reflex/models"
	"net/http"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/store.go --fake-name Store . store
type store interface {
	GetAddresses() []string
}

type PeersHandler struct {
	Logger lager.Logger
	Store  store
}

func (h *PeersHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var respData models.PeersResponse
	respData.IPs = h.Store.GetAddresses()
	err := json.NewEncoder(resp).Encode(respData)
	if err != nil {
		h.Logger.Error("json-encode", err)
		return
	}
	return
}
