package handlers

import (
	"net/http"
	"policy-server/api"
	"policy-server/uaa_client"

	"policy-server/store"

	"code.cloudfoundry.org/lager"
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

func (h *PoliciesIndex) ServeHTTP(logger lager.Logger, w http.ResponseWriter, req *http.Request, userToken uaa_client.CheckTokenResponse) {
	logger = logger.Session("index-policies")
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
		logger.Error("failed-reading-database", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-index", "database read failed")
		return
	}

	policies, err := h.PolicyFilter.FilterPolicies(storePolicies, userToken)
	if err != nil {
		logger.Error("failed-filtering-policies", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-index", "filter policies failed")
		return
	}

	for i, _ := range policies {
		policies[i].Source.Tag = ""
		policies[i].Destination.Tag = ""
	}

	bytes, err := h.Mapper.AsBytes(policies)
	if err != nil {
		logger.Error("failed-mapping-policies-as-bytes", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-index", "map policy as bytes failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
