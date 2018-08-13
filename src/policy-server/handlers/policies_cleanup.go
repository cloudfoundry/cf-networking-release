package handlers

import (
	"net/http"
	"policy-server/api"
	"policy-server/store"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/policy_cleaner.go --fake-name PolicyCleaner . policyCleaner
type policyCleaner interface {
	DeleteStalePolicies() (store.PolicyCollection, error)
}

//go:generate counterfeiter -o fakes/error_response.go --fake-name ErrorResponse . errorResponse
type errorResponse interface {
	InternalServerError(lager.Logger, http.ResponseWriter, error, string)
	BadRequest(lager.Logger, http.ResponseWriter, error, string)
	NotAcceptable(lager.Logger, http.ResponseWriter, error, string)
	Forbidden(lager.Logger, http.ResponseWriter, error, string)
	Unauthorized(lager.Logger, http.ResponseWriter, error, string)
}

type PoliciesCleanup struct {
	Mapper        api.PolicyMapper
	PolicyCleaner policyCleaner
	ErrorResponse errorResponse
}

func NewPoliciesCleanup(mapper api.PolicyMapper, policyCleaner policyCleaner, errorResponse errorResponse) *PoliciesCleanup {
	return &PoliciesCleanup{
		Mapper:        mapper,
		PolicyCleaner: policyCleaner,
		ErrorResponse: errorResponse,
	}
}

func (h *PoliciesCleanup) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("cleanup-policies")

	policyCollection, err := h.PolicyCleaner.DeleteStalePolicies()
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "policies cleanup failed")
		return
	}

	policies := policyCollection.Policies

	for i := range policies {
		policies[i].Source.Tag = ""
		policies[i].Destination.Tag = ""
	}

	bytes, err := h.Mapper.AsBytes(policies, policyCollection.EgressPolicies)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "map policy as bytes failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
