package handlers

import (
	"net/http"
	"net/url"
	"policy-server/api"
	"policy-server/store"
	"strings"

	"code.cloudfoundry.org/lager"
)

type PoliciesIndexInternal struct {
	Logger        lager.Logger
	Store         store.Store
	Mapper        api.PolicyMapper
	ErrorResponse errorResponse
	EgressStore   egressPolicyStore
	Conn          database
}

func NewPoliciesIndexInternal(logger lager.Logger, conn database, store store.Store, egressStore egressPolicyStore,
	mapper api.PolicyMapper, errorResponse errorResponse) *PoliciesIndexInternal {
	return &PoliciesIndexInternal{
		Logger:        logger,
		Conn:          conn,
		Store:         store,
		EgressStore:   egressStore,
		Mapper:        mapper,
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

	var egressPolicies []store.EgressPolicy
	if len(ids) == 0 {
		egressPolicies, err = h.EgressStore.AllWithTx()
	} else {
		egressPolicies, err = h.EgressStore.ByGuidsWithTx(ids)
	}

	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "egress database read failed")
		return
	}

	bytes, err := h.Mapper.AsBytes(policies, egressPolicies)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "map policy as bytes failed")
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
