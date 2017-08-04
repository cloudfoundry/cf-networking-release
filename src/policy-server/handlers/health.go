package handlers

import (
	"net/http"
)

type Health struct {
	Store         dataStore
	ErrorResponse errorResponse
}

func NewHealth(store dataStore, errorResponse errorResponse) *Health {
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
		logger.Error("failed-checking-database", err)
		h.ErrorResponse.InternalServerError(w, err, "health", "check database failed")
		return
	}
}
