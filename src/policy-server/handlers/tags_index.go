package handlers

import (
	"net/http"

	"policy-server/api"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
)

type TagsIndex struct {
	Store         dataStore
	Marshaler     marshal.Marshaler
	ErrorResponse errorResponse
}

func NewTagsIndex(store dataStore, marshaler marshal.Marshaler, errorResponse errorResponse) *TagsIndex {
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
		logger.Error("failed-reading-database", err)
		h.ErrorResponse.InternalServerError(w, err, "tags-index", "database read failed")
		return
	}

	tagsResponse := struct {
		Tags []api.Tag `json:"tags"`
	}{api.MapStoreTags(tags)}
	responseBytes, err := h.Marshaler.Marshal(tagsResponse)
	if err != nil {
		logger.Error("failed-marshalling-tags", err)
		h.ErrorResponse.InternalServerError(w, err, "tags-index", "database marshalling failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}
