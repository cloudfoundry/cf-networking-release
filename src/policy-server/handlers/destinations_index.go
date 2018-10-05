package handlers

import (
	"net/http"
	"policy-server/store"

	"code.cloudfoundry.org/lager"
)

type DestinationsIndex struct {
	ErrorResponse           errorResponse
	EgressDestinationStore  EgressDestinationStoreLister
	EgressDestinationMapper EgressDestinationMarshaller
	Logger                  lager.Logger
}

//go:generate counterfeiter -o fakes/egress_destination_marshaller.go --fake-name EgressDestinationMarshaller . EgressDestinationMarshaller
type EgressDestinationMarshaller interface {
	AsBytes(egressDestinations []store.EgressDestination) ([]byte, error)
	AsEgressDestinations([]byte) ([]store.EgressDestination, error)
}

//go:generate counterfeiter -o fakes/egress_destination_store_lister.go --fake-name EgressDestinationStoreLister . EgressDestinationStoreLister
type EgressDestinationStoreLister interface {
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
