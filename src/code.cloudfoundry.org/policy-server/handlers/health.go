package handlers

import (
	"net/http"

	"code.cloudfoundry.org/policy-server/store"
)

type Health struct {
	Store         store.Store
	ErrorResponse errorResponse
}

func NewHealth(store store.Store, errorResponse errorResponse) *Health {
	return &Health{
		Store:         store,
		ErrorResponse: errorResponse,
	}
}

func (h *Health) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("health")
	err := h.Store.CheckDatabase()
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "check database failed")
		return
	}
}
