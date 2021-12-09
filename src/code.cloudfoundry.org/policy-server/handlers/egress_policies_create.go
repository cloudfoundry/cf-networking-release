package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/policy-server/store"
)

//counterfeiter:generate -o fakes/egress_policy_mapper.go --fake-name EgressPolicyMapper . egressPolicyMapper
type egressPolicyMapper interface {
	AsStoreEgressPolicy(bytes []byte) ([]store.EgressPolicy, error)
	AsBytes(storeEgressPolicies []store.EgressPolicy) ([]byte, error)
	AsBytesWithPopulatedDestinations(storeEgressPolicies []store.EgressPolicy) ([]byte, error)
}

type EgressPolicyCreate struct {
	Store         egressPolicyStore
	Mapper        egressPolicyMapper
	ErrorResponse errorResponse
	Logger        lager.Logger
}

func (e *EgressPolicyCreate) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	requestBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		e.ErrorResponse.BadRequest(e.Logger, w, err, "error reading request")
		return
	}

	storeEgressPolicies, err := e.Mapper.AsStoreEgressPolicy(requestBytes)
	if err != nil {
		e.ErrorResponse.BadRequest(e.Logger, w, err, fmt.Sprintf("error parsing egress policies: %s", err))
		return
	}

	createdPolicies, err := e.Store.Create(storeEgressPolicies)
	if err != nil {
		e.ErrorResponse.InternalServerError(e.Logger, w, err, "error creating egress policy")
		return
	}

	bytes, err := e.Mapper.AsBytes(createdPolicies)
	if err != nil {
		e.ErrorResponse.InternalServerError(e.Logger, w, err, "error serializing response")
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(bytes)
}
