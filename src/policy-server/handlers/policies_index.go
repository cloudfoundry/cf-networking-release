package handlers

import (
	"net/http"
	"net/url"
	"policy-server/api"
	"policy-server/uaa_client"
	"strings"

	"policy-server/db"
	"policy-server/store"
)

//go:generate counterfeiter -o fakes/policy_filter.go --fake-name PolicyFilter . policyFilter
type policyFilter interface {
	FilterPolicies(policies []store.Policy, userToken uaa_client.CheckTokenResponse) ([]store.Policy, error)
}

//go:generate counterfeiter -o fakes/egress_policy_store.go --fake-name EgressPolicyStore . egressPolicyStore
type egressPolicyStore interface {
	All() ([]store.EgressPolicy, error)
	ByGuids(ids []string) ([]store.EgressPolicy, error)
	Create(egressPolicies []store.EgressPolicy) error
}

//go:generate counterfeiter -o fakes/database.go --fake-name Db . database
type database interface {
	Beginx() (db.Transaction, error)
}

type PoliciesIndex struct {
	Store         store.Store
	Mapper        api.PolicyMapper
	PolicyFilter  policyFilter
	PolicyGuard   policyGuard
	ErrorResponse errorResponse
}

func NewPoliciesIndex(store store.Store,
	mapper api.PolicyMapper, policyFilter policyFilter, policyGuard policyGuard, errorResponse errorResponse) *PoliciesIndex {
	return &PoliciesIndex{
		Store:         store,
		Mapper:        mapper,
		PolicyFilter:  policyFilter,
		PolicyGuard:   policyGuard,
		ErrorResponse: errorResponse,
	}
}

func (h *PoliciesIndex) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("index-policies")
	userToken := getTokenData(req)
	queryValues := req.URL.Query()
	ids := parseIds(queryValues)
	sourceIDs := parseSourceIds(queryValues)
	destIDs := parseDestIds(queryValues)

	var storePolicies []store.Policy
	var err error
	if len(ids) > 0 {
		storePolicies, err = h.Store.ByGuids(ids, ids, false)
	} else if len(sourceIDs) > 0 && len(destIDs) > 0 {
		storePolicies, err = h.Store.ByGuids(sourceIDs, destIDs, true)
	} else if len(sourceIDs) > 0 {
		storePolicies, err = h.Store.ByGuids(sourceIDs, []string{}, false)
	} else if len(destIDs) > 0 {
		storePolicies, err = h.Store.ByGuids([]string{}, destIDs, false)
	} else {
		storePolicies, err = h.Store.All()
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

	for i := range policies {
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

func parseSourceIds(queryValues url.Values) []string {
	var ids []string
	idList, ok := queryValues["source_id"]
	if ok {
		ids = strings.Split(idList[0], ",")
	}
	return ids
}

func parseDestIds(queryValues url.Values) []string {
	var ids []string
	idList, ok := queryValues["dest_id"]
	if ok {
		ids = strings.Split(idList[0], ",")
	}
	return ids
}
