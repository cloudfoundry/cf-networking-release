package handlers

import (
	"net/http"
	"net/url"
	"policy-server/models"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/store.go --fake-name Store . store
type store interface {
	All() ([]models.Policy, error)
	Create([]models.Policy) error
	Delete([]models.Policy) error
	Tags() ([]models.Tag, error)
	ByGuids([]string, []string) ([]models.Policy, error)
	CheckDatabase() error
}

type PoliciesIndexInternal struct {
	Logger        lager.Logger
	Store         store
	Marshaler     marshal.Marshaler
	ErrorResponse errorResponse
}

func (h *PoliciesIndexInternal) ServeHTTP(logger lager.Logger, w http.ResponseWriter, req *http.Request) {
	logger = logger.Session("index-policies-internal")

	queryValues := req.URL.Query()
	ids := parseIds(queryValues)

	var policies []models.Policy
	var err error
	if len(ids) == 0 {
		policies, err = h.Store.All()
	} else {
		policies, err = h.Store.ByGuids(ids, ids)
	}

	if err != nil {
		logger.Error("failed-reading-database", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-index-internal", "database read failed")
		return
	}

	policyResponse := struct {
		Policies []models.Policy `json:"policies"`
	}{policies}
	bytes, err := h.Marshaler.Marshal(policyResponse)
	if err != nil {
		logger.Error("failed-marshalling-policies", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-index-internal", "database marshalling failed")
		return
	}

	w.Write(bytes)
}

func parseIds(queryValues url.Values) []string {
	var ids []string
	idList, ok := queryValues["id"]
	if ok {
		ids = strings.Split(idList[0], ",")
	}
	return ids
}
