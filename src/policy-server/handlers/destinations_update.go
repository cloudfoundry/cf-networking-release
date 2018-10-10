package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"policy-server/store"
	"strings"

	"code.cloudfoundry.org/lager"
)

type DestinationsUpdate struct {
	ErrorResponse           errorResponse
	EgressDestinationStore  EgressDestinationStoreUpdater
	EgressDestinationMapper EgressDestinationMarshaller
	Logger                  lager.Logger
}

//go:generate counterfeiter -o fakes/egress_destination_store_updater.go --fake-name EgressDestinationStoreUpdater . EgressDestinationStoreUpdater
type EgressDestinationStoreUpdater interface {
	Update([]store.EgressDestination) ([]store.EgressDestination, error)
}

func (d *DestinationsUpdate) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var destinations, updatedDestinations []store.EgressDestination
	var requestBytes, responseBytes []byte
	var err error

	requestBytes, err = ioutil.ReadAll(req.Body)
	if err != nil {
		d.ErrorResponse.InternalServerError(d.Logger, w, err, "error reading request")
		return
	}

	destinations, err = d.EgressDestinationMapper.AsEgressDestinations(requestBytes)
	if err != nil {
		d.ErrorResponse.BadRequest(d.Logger, w, err, fmt.Sprintf("error parsing egress destination: %s", err))
		return
	}

	for _, destination := range destinations {
		if destination.GUID == "" {
			d.ErrorResponse.BadRequest(d.Logger, w, nil, fmt.Sprintf("destination id not found on request"))
			return
		}
	}

	updatedDestinations, err = d.EgressDestinationStore.Update(destinations)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate name error") {
			d.ErrorResponse.BadRequest(d.Logger, w, err, fmt.Sprintf("error updating egress destination: %s", err))
			return
		}
		if strings.Contains(err.Error(), "destination GUID not found") {
			d.ErrorResponse.NotFound(d.Logger, w, err, fmt.Sprintf("error updating egress destination: %s", err))
			return
		}
		d.ErrorResponse.InternalServerError(d.Logger, w, err, "error updating egress destination")
		return
	}
	responseBytes, err = d.EgressDestinationMapper.AsBytes(updatedDestinations)
	if err != nil {
		d.ErrorResponse.InternalServerError(d.Logger, w, err, "error serializing egress destinations")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}
