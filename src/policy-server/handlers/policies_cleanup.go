package handlers

import (
	"net/http"
	"policy-server/models"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/go-db-helpers/marshal"
	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/policy_cleaner.go --fake-name PolicyCleaner . policyCleaner
type policyCleaner interface {
	DeleteStalePolicies() ([]models.Policy, error)
}

//go:generate counterfeiter -o fakes/error_response.go --fake-name ErrorResponse . errorResponse
type errorResponse interface {
	InternalServerError(http.ResponseWriter, error, string, string)
	BadRequest(http.ResponseWriter, error, string, string)
	Forbidden(http.ResponseWriter, error, string, string)
	Unauthorized(http.ResponseWriter, error, string, string)
}

type PoliciesCleanup struct {
	Logger        lager.Logger
	Marshaler     marshal.Marshaler
	PolicyCleaner policyCleaner
	ErrorResponse errorResponse
}

func (h *PoliciesCleanup) ServeHTTP(w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
	policies, err := h.PolicyCleaner.DeleteStalePolicies()
	if err != nil {
		h.ErrorResponse.InternalServerError(w, err, "policies-cleanup", "policies cleanup failed")
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
		h.ErrorResponse.InternalServerError(w, err, "policies-cleanup", "marshal response failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
