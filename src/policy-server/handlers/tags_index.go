package handlers

import (
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager"
)

type TagsIndex struct {
	Logger    lager.Logger
	Store     store
	Marshaler marshal.Marshaler
}

func (h *TagsIndex) ServeHTTP(w http.ResponseWriter, req *http.Request, _ uaa_client.CheckTokenResponse) {
	tags, err := h.Store.Tags()
	if err != nil {
		h.Logger.Error("store-list-tags-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database read failed"}`))
		return
	}

	tagsResponse := struct {
		Tags []models.Tag `json:"tags"`
	}{tags}
	responseBytes, err := h.Marshaler.Marshal(tagsResponse)
	if err != nil {
		h.Logger.Error("marshal-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database marshaling failed"}`))
		return
	}
	w.Write(responseBytes)
}
