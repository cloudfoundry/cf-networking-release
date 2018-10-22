package handlers

import (
	"net/http"
	"policy-server/store"

	"net/url"
	"strings"

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
	GetByGUID(guid ...string) ([]store.EgressDestination, error)
	GetByName(name ...string) ([]store.EgressDestination, error)
}

func (d *DestinationsIndex) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	queryParameters := req.URL.Query()
	guid := parseQueryParam(queryParameters, "id")
	name := parseQueryParam(queryParameters, "name")

	var egressDestinations []store.EgressDestination
	var err error

	if len(guid) > 0 {
		egressDestinations, err = d.EgressDestinationStore.GetByGUID(guid...)
	} else if len(name) > 0 {
		egressDestinations, err = d.EgressDestinationStore.GetByName(name...)
	} else {
		egressDestinations, err = d.EgressDestinationStore.All()
	}

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

func parseQueryParam(queryValues url.Values, queryParam string) []string {
	var values []string
	v, ok := queryValues[queryParam]
	if ok {
		values = strings.Split(v[0], ",")
	}
	return values
}
