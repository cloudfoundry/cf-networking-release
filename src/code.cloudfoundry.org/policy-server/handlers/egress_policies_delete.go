package handlers

import (
	"net/http"

	"code.cloudfoundry.org/lager"
)

type EgressPolicyDelete struct {
	Store         egressPolicyStore
	Mapper        egressPolicyMapper
	ErrorResponse errorResponse
	Logger        lager.Logger
}

func (e *EgressPolicyDelete) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	guid := req.URL.Query().Get(":id")

	deletedPolicies, err := e.Store.Delete(guid)
	if err != nil {
		e.ErrorResponse.InternalServerError(e.Logger, w, err, "error deleting egress policy")
		return
	}

	bytes, err := e.Mapper.AsBytes(deletedPolicies)
	if err != nil {
		e.ErrorResponse.InternalServerError(e.Logger, w, err, "error serializing response")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
