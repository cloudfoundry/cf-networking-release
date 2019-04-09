package handlers

import (
	"net/http"
)

type XXSSProtectionHandler struct {
}

func (x XXSSProtectionHandler) Wrap(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		handler.ServeHTTP(w, req)
	})
}
