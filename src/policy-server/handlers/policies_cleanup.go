package handlers

import (
	"net/http"
	"policy-server/api"
	"policy-server/store"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/policy_cleaner.go --fake-name PolicyCleaner . policyCleaner
type policyCleaner interface {
	DeleteStalePolicies() ([]store.Policy, []store.EgressPolicy, error)
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
	PolicyCollectionWriter api.PolicyCollectionWriter
	PolicyCleaner          policyCleaner
	ErrorResponse          errorResponse
}

func NewPoliciesCleanup(writer api.PolicyCollectionWriter, policyCleaner policyCleaner, errorResponse errorResponse) *PoliciesCleanup {
	return &PoliciesCleanup{
		PolicyCollectionWriter: writer,
		PolicyCleaner:          policyCleaner,
		ErrorResponse:          errorResponse,
	}
}

func (h *PoliciesCleanup) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("cleanup-policies")

	c2cPolicies, egressPolicies, err := h.PolicyCleaner.DeleteStalePolicies()
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "policies cleanup failed")
		return
	}

	for i := range c2cPolicies {
		c2cPolicies[i].Source.Tag = ""
		c2cPolicies[i].Destination.Tag = ""
	}

	bytes, err := h.PolicyCollectionWriter.AsBytes(c2cPolicies, egressPolicies)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "map policy as bytes failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
