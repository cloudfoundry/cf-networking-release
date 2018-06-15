package handlers

import (
	"net/http"

	"policy-server/api"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"policy-server/store"
)

type TagsIndex struct {
	Store         store.TagStore
	Marshaler     marshal.Marshaler
	ErrorResponse errorResponse
}

func NewTagsIndex(store store.TagStore, marshaler marshal.Marshaler, errorResponse errorResponse) *TagsIndex {
	return &TagsIndex{
		Store:         store,
		Marshaler:     marshaler,
		ErrorResponse: errorResponse,
	}
}

func (h *TagsIndex) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("index-tags")
	tags, err := h.Store.Tags()
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "database read failed")
		return
	}

	tagsResponse := struct {
		Tags []api.Tag `json:"tags"`
	}{api.MapStoreTags(tags)}
	responseBytes, err := h.Marshaler.Marshal(tagsResponse)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "database marshalling failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}
