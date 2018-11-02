package handlers

import (
	"net/http"
	"policy-server/store"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/egress_destination_store_deleter.go --fake-name EgressDestinationStoreDeleter . EgressDestinationStoreDeleter
type EgressDestinationStoreDeleter interface {
	Delete(...string) ([]store.EgressDestination, error)
}

type DestinationDelete struct {
	ErrorResponse           errorResponse
	EgressDestinationStore  EgressDestinationStoreDeleter
	EgressDestinationMapper EgressDestinationMarshaller
	Logger                  lager.Logger
}

func (d *DestinationDelete) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	guid := req.URL.Query().Get(":id")
	logger := getLogger(req)

	deletedDestinations, err := d.EgressDestinationStore.Delete(guid)
	if err != nil {
		switch err.(type) {
		case store.ForeignKeyError:
			d.ErrorResponse.BadRequest(logger, w, err, "destination is still in use")
			return
		default:
			d.ErrorResponse.InternalServerError(logger, w, err, "error deleting egress destination")
			return
		}
	}

	responseBody, err := d.EgressDestinationMapper.AsBytes(deletedDestinations)
	if err != nil {
		d.ErrorResponse.InternalServerError(logger, w, err, "error serializing egress destination")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseBody)
}
