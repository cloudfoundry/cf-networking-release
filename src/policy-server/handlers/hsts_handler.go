package handlers

import (
	"net/http"
)

type HSTSHandler struct {
}

func (x HSTSHandler) Wrap(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		handler.ServeHTTP(w, req)
	})
}
