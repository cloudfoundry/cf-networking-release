package handlers

import (
	"fmt"
	"net/http"
	"time"
)

const oneYear = time.Hour * 24 * 365

type HSTSHandler struct {
}

func (x HSTSHandler) Wrap(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Strict-Transport-Security", fmt.Sprintf("max-age=%.0f", oneYear.Seconds()))
		handler.ServeHTTP(w, req)
	})
}
