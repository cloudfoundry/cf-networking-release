package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/policy-server/store"
)

type DestinationsCreate struct {
	ErrorResponse           errorResponse
	EgressDestinationStore  EgressDestinationStoreCreator
	EgressDestinationMapper EgressDestinationMarshaller
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
		if strings.Contains(err.Error(), "duplicate name error") {
			d.ErrorResponse.BadRequest(d.Logger, w, err, fmt.Sprintf("error creating egress destinations: %s", err))
			return
		}
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
