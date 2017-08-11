package handlers

import (
	"net/http"
	"policy-server/api"
	"policy-server/uaa_client"

	"policy-server/store"
)

//go:generate counterfeiter -o fakes/policy_filter.go --fake-name PolicyFilter . policyFilter
type policyFilter interface {
	FilterPolicies(policies []store.Policy, userToken uaa_client.CheckTokenResponse) ([]store.Policy, error)
}

type PoliciesIndex struct {
	Store         dataStore
	Mapper        api.PolicyMapper
	PolicyFilter  policyFilter
	ErrorResponse errorResponse
}

func NewPoliciesIndex(store dataStore, mapper api.PolicyMapper, policyFilter policyFilter,
	errorResponse errorResponse) *PoliciesIndex {
	return &PoliciesIndex{
		Store:         store,
		Mapper:        mapper,
		PolicyFilter:  policyFilter,
		ErrorResponse: errorResponse,
	}
}

func (h *PoliciesIndex) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("index-policies")
	userToken := getTokenData(req)
	queryValues := req.URL.Query()
	ids := parseIds(queryValues)

	var storePolicies []store.Policy
	var err error
	if len(ids) == 0 {
		storePolicies, err = h.Store.All()
	} else {
		storePolicies, err = h.Store.ByGuids(ids, ids)
	}

	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "database read failed")
		return
	}

	policies, err := h.PolicyFilter.FilterPolicies(storePolicies, userToken)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "filter policies failed")
		return
	}

	for i, _ := range policies {
		policies[i].Source.Tag = ""
		policies[i].Destination.Tag = ""
	}

	bytes, err := h.Mapper.AsBytes(policies)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "map policy as bytes failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
