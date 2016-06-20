package handlers

import (
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/store"
	"strings"

	"github.com/pivotal-golang/lager"
)

type PoliciesIndex struct {
	Logger    lager.Logger
	Store     store.Store
	Marshaler marshal.Marshaler
}

func (h *PoliciesIndex) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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

	policyResponse := struct {
		Policies []models.Policy `json:"policies"`
	}{policies}
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
