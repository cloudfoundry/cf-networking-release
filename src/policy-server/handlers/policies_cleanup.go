package handlers

import (
	"net/http"
	"policy-server/api"
	"policy-server/store"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/policy_cleaner.go --fake-name PolicyCleaner . policyCleaner
type policyCleaner interface {
	DeleteStalePolicies() ([]store.Policy, error)
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
	Mapper        api.PolicyMapper
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

	for i, _ := range policies {
		policies[i].Source.Tag = ""
		policies[i].Destination.Tag = ""
	}

	bytes, err := h.Mapper.AsBytes(policies)
	if err != nil {
		logger.Error("failed-mapping-policies-as-bytes", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-cleanup", "map policy as bytes failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
