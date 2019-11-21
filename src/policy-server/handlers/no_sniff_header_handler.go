package handlers

import "net/http"

type NoSniffHeaderHandler struct {
}

func (n NoSniffHeaderHandler) Wrap(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		handler.ServeHTTP(w, req)
	})
}
