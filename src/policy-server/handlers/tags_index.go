package handlers

import (
	"net/http"
	"policy-server/models"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/lager"
)

type TagsIndex struct {
	Logger        lager.Logger
	Store         store
	Marshaler     marshal.Marshaler
	ErrorResponse errorResponse
}

func (h *TagsIndex) ServeHTTP(w http.ResponseWriter, req *http.Request, _ uaa_client.CheckTokenResponse) {
	tags, err := h.Store.Tags()
	if err != nil {
		h.ErrorResponse.InternalServerError(w, err, "tags-index", "database read failed")
		return
	}

	tagsResponse := struct {
		Tags []models.Tag `json:"tags"`
	}{tags}
	responseBytes, err := h.Marshaler.Marshal(tagsResponse)
	if err != nil {
		h.ErrorResponse.InternalServerError(w, err, "tags-index", "database marshaling failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}
