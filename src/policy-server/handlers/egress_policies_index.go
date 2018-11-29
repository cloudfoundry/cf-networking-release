package handlers

import (
	"net/http"
	"policy-server/store"

	"code.cloudfoundry.org/lager"
)

type EgressPolicyIndex struct {
	Store         egressPolicyStore
	Mapper        egressPolicyMapper
	ErrorResponse errorResponse
	Logger        lager.Logger
}

func (e *EgressPolicyIndex) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	queryParameters := req.URL.Query()
	sourceIds := parseQueryParam(queryParameters, "SourceIDs")
	sourceTypes := parseQueryParam(queryParameters, "SourceTypes")
	destinationIds := parseQueryParam(queryParameters, "DestinationIDs")
	destinationNames := parseQueryParam(queryParameters, "DestinationNames")

	var policies []store.EgressPolicy
	var err error
	policies, err = e.Store.GetByFilter(sourceIds, sourceTypes, destinationIds, destinationNames, []string{})
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
