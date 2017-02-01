package handlers

import (
	"fmt"
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/policy_cleaner.go --fake-name PolicyCleaner . policyCleaner
type policyCleaner interface {
	DeleteStalePolicies() ([]models.Policy, error)
}

type PoliciesCleanup struct {
	Logger        lager.Logger
	Marshaler     marshal.Marshaler
	PolicyCleaner policyCleaner
}

func (h *PoliciesCleanup) ServeHTTP(w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
	policies, err := h.PolicyCleaner.DeleteStalePolicies()
	if err != nil {
		h.Logger.Error("policies-cleanup", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "policies cleanup failed"}`))
		return
	}
	policyCleanup := struct {
		TotalPolicies int             `json:"total_policies"`
		Policies      []models.Policy `json:"policies"`
	}{len(policies), policies}
	for i, _ := range policyCleanup.Policies {
		policyCleanup.Policies[i].Source.Tag = ""
		policyCleanup.Policies[i].Destination.Tag = ""
	}

	bytes, err := h.Marshaler.Marshal(policyCleanup)
	if err != nil {
		h.Logger.Error("marshal-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"error": "marshal response failed"}`)))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
