package handlers

import (
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/store"
	"strings"

	"code.cloudfoundry.org/lager"
)

type PoliciesIndex struct {
	Logger    lager.Logger
	Store     store.Store
	Marshaler marshal.Marshaler
}

func (h *PoliciesIndex) ServeHTTP(w http.ResponseWriter, req *http.Request, currentUserName string) {
	policies, err := h.Store.All()
	if err != nil {
		h.Logger.Error("store-list-policies-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database read failed"}`))
		return
	}

	queryValues := req.URL.Query()
	idList, ok := queryValues["id"]
	if ok {
		ids := strings.Split(idList[0], ",")
		policies = filterByID(policies, ids)
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
		h.Logger.Error("marshal-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database marshaling failed"}`))
		return
	}
	w.Write(bytes)
}

func filterByID(policies []models.Policy, ids []string) []models.Policy {
	filteredPolicies := []models.Policy{}
	for _, policy := range policies {
		if containsID(policy, ids) {
			filteredPolicies = append(filteredPolicies, policy)
		}
	}
	return filteredPolicies
}

func containsID(policy models.Policy, ids []string) bool {
	for _, id := range ids {
		if id == policy.Source.ID || id == policy.Destination.ID {
			return true
		}
	}
	return false
}
