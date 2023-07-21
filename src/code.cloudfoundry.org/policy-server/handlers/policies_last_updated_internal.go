package handlers

import (
	"net/http"
	"strconv"

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/policy-server/store"
)

type PoliciesLastUpdatedInternal struct {
	Logger        lager.Logger
	Store         store.Store
	ErrorResponse errorResponse
}

func NewPoliciesLastUpdatedInternal(logger lager.Logger, store store.Store,
	errorResponse errorResponse) *PoliciesLastUpdatedInternal {
	return &PoliciesLastUpdatedInternal{
		Logger:        logger,
		Store:         store,
		ErrorResponse: errorResponse,
	}
}

func (h *PoliciesLastUpdatedInternal) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("policies-last-updated-internal")

	lastUpdated, err := h.Store.LastUpdated()
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "database read failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strconv.Itoa(lastUpdated)))
}
