package handlers

import (
	"net/http"
	"strconv"

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/policy-server/store"
)

type SecurityGroupsLastUpdatedInternal struct {
	Logger        lager.Logger
	Store         store.SecurityGroupsStore
	ErrorResponse errorResponse
}

func NewSecurityGroupsLastUpdatedInternal(logger lager.Logger, store store.SecurityGroupsStore,
	errorResponse errorResponse) *SecurityGroupsLastUpdatedInternal {
	return &SecurityGroupsLastUpdatedInternal{
		Logger:        logger,
		Store:         store,
		ErrorResponse: errorResponse,
	}
}

func (h *SecurityGroupsLastUpdatedInternal) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("security-groups-last-updated-internal")

	lastUpdated, err := h.Store.LastUpdated()
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "database read failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	// #nosec G104 - ignore errors writing http responses to avoid spamming logs during a DoS
	w.Write([]byte(strconv.Itoa(lastUpdated)))
}
