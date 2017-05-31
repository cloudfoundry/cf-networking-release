package handlers

import "net/http"

type Health struct {
	Store         store
	ErrorResponse errorResponse
}

func (h *Health) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	err := h.Store.CheckDatabase()
	if err != nil {
		h.ErrorResponse.InternalServerError(w, err, "health", "check database failed")
		return
	}
}
