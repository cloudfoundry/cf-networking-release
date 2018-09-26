package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"policy-server/store"

	"code.cloudfoundry.org/lager"
)

type DestinationsCreate struct {
	ErrorResponse           errorResponse
	EgressDestinationStore  EgressDestinationStoreCreator
	EgressDestinationMapper EgressDestinationMarshaller
	PolicyGuard             policyGuard
	Logger                  lager.Logger
}

//go:generate counterfeiter -o fakes/egress_destination_store_creator.go --fake-name EgressDestinationStoreCreator . EgressDestinationStoreCreator
type EgressDestinationStoreCreator interface {
	Create([]store.EgressDestination) ([]store.EgressDestination, error)
}

func (d *DestinationsCreate) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var destinations, createdDestinations []store.EgressDestination
	var requestBytes, responseBytes []byte
	var err error

	userToken := getTokenData(req)
	if policyGuard.IsNetworkAdmin(d.PolicyGuard, userToken) == false {
		d.ErrorResponse.Forbidden(d.Logger, w, err, "not authorized: creating egress destinations failed")
		return
	}

	requestBytes, err = ioutil.ReadAll(req.Body)
	if err != nil {
		d.ErrorResponse.InternalServerError(d.Logger, w, err, "error reading request")
		return
	}
	destinations, err = d.EgressDestinationMapper.AsEgressDestinations(requestBytes)
	if err != nil {
		d.ErrorResponse.BadRequest(d.Logger, w, err, fmt.Sprintf("error parsing egress destinations: %s", err))
		return
	}
	createdDestinations, err = d.EgressDestinationStore.Create(destinations)
	if err != nil {
		d.ErrorResponse.InternalServerError(d.Logger, w, err, "error creating egress destinations")
		return
	}
	responseBytes, err = d.EgressDestinationMapper.AsBytes(createdDestinations)
	if err != nil {
		d.ErrorResponse.InternalServerError(d.Logger, w, err, "error serializing egress destinations")
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(responseBytes)
}
