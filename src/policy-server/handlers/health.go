package handlers

import (
	"net/http"

	"code.cloudfoundry.org/lager"
)

type Health struct {
	Store         store
	ErrorResponse errorResponse
}

func (h *Health) ServeHTTP(logger lager.Logger, w http.ResponseWriter, req *http.Request) {
	logger = logger.Session("health")
	err := h.Store.CheckDatabase()
	if err != nil {
		logger.Error("failed-checking-database", err)
		h.ErrorResponse.InternalServerError(w, err, "health", "check database failed")
		return
	}
}
