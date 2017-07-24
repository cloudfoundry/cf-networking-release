package handlers

import (
	"net/http"
	"policy-server/api"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/policy_cleaner.go --fake-name PolicyCleaner . policyCleaner
type policyCleaner interface {
	DeleteStalePolicies() ([]api.Policy, error)
}

//go:generate counterfeiter -o fakes/error_response.go --fake-name ErrorResponse . errorResponse
type errorResponse interface {
	InternalServerError(http.ResponseWriter, error, string, string)
	BadRequest(http.ResponseWriter, error, string, string)
	NotAcceptable(http.ResponseWriter, error, string, string)
	Forbidden(http.ResponseWriter, error, string, string)
	Unauthorized(http.ResponseWriter, error, string, string)
}

type PoliciesCleanup struct {
	Marshaler     marshal.Marshaler
	PolicyCleaner policyCleaner
	ErrorResponse errorResponse
}

func (h *PoliciesCleanup) ServeHTTP(logger lager.Logger, w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
	logger = logger.Session("cleanup-policies")
	policies, err := h.PolicyCleaner.DeleteStalePolicies()
	if err != nil {
		logger.Error("failed-deleting-stale-policies", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-cleanup", "policies cleanup failed")
		return
	}

	policyCleanup := struct {
		TotalPolicies int             `json:"total_policies"`
		Policies      []api.Policy `json:"policies"`
	}{len(policies), policies}
	for i, _ := range policyCleanup.Policies {
		policyCleanup.Policies[i].Source.Tag = ""
		policyCleanup.Policies[i].Destination.Tag = ""
	}

	bytes, err := h.Marshaler.Marshal(policyCleanup)
	if err != nil {
		logger.Error("failed-marshalling-policies", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-cleanup", "marshal response failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
