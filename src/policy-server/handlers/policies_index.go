package handlers

import (
	"net/http"
	"policy-server/models"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/policy_filter.go --fake-name PolicyFilter . policyFilter
type policyFilter interface {
	FilterPolicies(policies []models.Policy, userToken uaa_client.CheckTokenResponse) ([]models.Policy, error)
}

type PoliciesIndex struct {
	Store         store
	Marshaler     marshal.Marshaler
	PolicyFilter  policyFilter
	ErrorResponse errorResponse
}

func (h *PoliciesIndex) ServeHTTP(logger lager.Logger, w http.ResponseWriter, req *http.Request, userToken uaa_client.CheckTokenResponse) {
	logger = logger.Session("index-policies")
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
		h.ErrorResponse.InternalServerError(w, err, "policies-index", "database read failed")
		return
	}

	policies, err = h.PolicyFilter.FilterPolicies(policies, userToken)
	if err != nil {
		logger.Error("failed-filtering-policies", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-index", "filter policies failed")
		return
	}

	for i, _ := range policies {
		policies[i].Source.Tag = ""
		policies[i].Destination.Tag = ""
	}

	policyResponse := struct {
		TotalPolicies int             `json:"total_policies"`
		Policies      []models.Policy `json:"policies"`
	}{len(policies), policies}
	bytes, err := h.Marshaler.Marshal(policyResponse)
	if err != nil {
		logger.Error("failed-marshalling-policies", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-index", "database marshalling failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
