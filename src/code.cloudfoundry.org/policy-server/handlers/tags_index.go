package handlers

import (
	"net/http"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/policy-server/api"
	"code.cloudfoundry.org/policy-server/store"
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

// @Summary List tags
// @Description Retrieves all tag and ID mappings. Tags are unique identifiers assigned to policy groups for network policy enforcement.
// @Tags tags
// @Produce json
// @Success 200 {object} object{tags=[]api.Tag} "Successfully retrieved tags"
// @Failure 500 {object} httperror.ErrorResponse "Internal server error"
// @Router /tags [get]
// @Security OAuth2Application[network.admin]
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
	// #nosec G104 - ignore errors writing http responses to avoid spamming logs during a DoS
	w.Write(responseBytes)
}
