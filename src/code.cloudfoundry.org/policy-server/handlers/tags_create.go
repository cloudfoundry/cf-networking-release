package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"code.cloudfoundry.org/policy-server/api"
	"code.cloudfoundry.org/policy-server/store"
)

//counterfeiter:generate -o fakes/create_tag_store.go --fake-name CreateTagDataStore . createTagDataStore
type createTagDataStore interface {
	CreateTag(string, string) (store.Tag, error)
}

type TagsCreate struct {
	Store         createTagDataStore
	ErrorResponse errorResponse
}

type Group struct {
	GroupType string `json:"type"`
	GroupGuid string `json:"id"`
}

func (h *TagsCreate) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("create-tags")

	bodyBytes, err := io.ReadAll(req.Body)

	if err != nil {
		h.ErrorResponse.BadRequest(logger, w, err, "failed reading request body")
		return
	}

	var grp Group
	err = json.Unmarshal(bodyBytes, &grp)
	if err != nil {
		h.ErrorResponse.BadRequest(logger, w, err, "failed parsing request body")
		return
	}

	tag, err := h.Store.CreateTag(grp.GroupGuid, grp.GroupType)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "database create failed")
		return
	}

	tagJSON, err := json.Marshal(api.MapStoreTag(tag))
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, fmt.Sprintf("failed to marshal tag with id: %s, type: %s, and tag: %s", tag.ID, tag.Type, tag.Tag))
		return
	}
	w.WriteHeader(http.StatusOK)
	// #nosec G104 - ignore errors writing http responses to avoid spamming logs during a DoS
	w.Write(tagJSON)
}
