package handlers

import (
	"net/http"
	"policy-server/store"
	"code.cloudfoundry.org/lager"
)

type DestinationsIndex struct {
	ErrorResponse errorResponse
	EgressDestinationStore EgressDestinationStore
	EgressDestinationMapper EgressDestinationMapper
	Logger lager.Logger
}

//go:generate counterfeiter -o fakes/egress_destination_mapper.go --fake-name EgressDestinationMapper . EgressDestinationMapper
type EgressDestinationMapper interface {
	AsBytes(egressDestinations []store.EgressDestination) ([]byte, error)
}

//go:generate counterfeiter -o fakes/egress_destination_store.go --fake-name EgressDestinationStore . EgressDestinationStore
type EgressDestinationStore interface {
	All() ([]store.EgressDestination, error)
}

func (d *DestinationsIndex) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	egressDestinations, err := d.EgressDestinationStore.All()
	if err != nil {
		d.ErrorResponse.InternalServerError(d.Logger, w, err, "error getting egress destinations")
		return
	}
	responseBytes, err := d.EgressDestinationMapper.AsBytes(egressDestinations)
	if err != nil {
		d.ErrorResponse.InternalServerError(d.Logger, w, err, "error mapping egress destinations")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

