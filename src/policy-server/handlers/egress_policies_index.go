package handlers

import (
	"net/http"

	"code.cloudfoundry.org/lager"
)

type EgressPolicyIndex struct {
	Store         egressPolicyStore
	Mapper        egressPolicyMapper
	ErrorResponse errorResponse
	Logger        lager.Logger
}

func (e *EgressPolicyIndex) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	policies, err := e.Store.All()
	if err != nil {
		e.ErrorResponse.InternalServerError(e.Logger, w, err, "error listing egress policies")
		return
	}

	bytes, err := e.Mapper.AsBytesWithPopulatedDestinations(policies)
	if err != nil {
		e.ErrorResponse.InternalServerError(e.Logger, w, err, "error serializing response")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}
