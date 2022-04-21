package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/policy-server/api"
	"code.cloudfoundry.org/policy-server/store"
)

type PoliciesIndexInternal struct {
	Logger        lager.Logger
	Store         store.Store
	PolicyMapper  api.PolicyMapper
	ErrorResponse errorResponse
}

func NewPoliciesIndexInternal(logger lager.Logger, store store.Store, writer api.PolicyMapper,
	errorResponse errorResponse) *PoliciesIndexInternal {
	return &PoliciesIndexInternal{
		Logger:        logger,
		Store:         store,
		PolicyMapper:  writer,
		ErrorResponse: errorResponse,
	}
}

func (h *PoliciesIndexInternal) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("index-policies-internal")

	queryValues := req.URL.Query()
	ids := parseIds(queryValues)

	var policies []store.Policy
	var err error
	if len(ids) == 0 {
		policies, err = h.Store.All()
	} else {
		policies, err = h.Store.ByGuids(ids, ids, false)
	}

	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "database read failed")
		return
	}

	bytes, err := h.PolicyMapper.AsBytes(policies)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "map policies as bytes failed")
		return
	}

	w.WriteHeader(http.StatusOK)
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
