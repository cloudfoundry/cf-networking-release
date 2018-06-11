package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

//go:generate counterfeiter -o fakes/create_tag_store.go --fake-name CreateTagDataStore . createTagDataStore
type createTagDataStore interface {
	CreateTag(string, string) (string, error)
}

type TagsCreate struct {
	Store         createTagDataStore
	ErrorResponse errorResponse
}

type Group struct {
	GroupType string `json:"type"`
	GroupGuid string `json:"guid"`
}

func (h *TagsCreate) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("create-tags")

	bodyBytes, err := ioutil.ReadAll(req.Body)

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

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{ "tag": "%s" }`, tag)))
}
