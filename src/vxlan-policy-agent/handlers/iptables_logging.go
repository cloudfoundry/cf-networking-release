package handlers

import (
	"encoding/json"
	"net/http"
)

//go:generate counterfeiter -o ../fakes/loggingState.go --fake-name LoggingState . loggingState
type loggingState interface {
	Disable()
	Enable()
	IsEnabled() bool
}

type IPTablesLogging struct {
	LoggingState loggingState
}

func (h *IPTablesLogging) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		var bodyStruct = struct {
			Enabled *bool `json:"enabled"`
		}{}

		err := json.NewDecoder(r.Body).Decode(&bodyStruct)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{ "error": "decoding request body as json" }`))
			return
		}
		if bodyStruct.Enabled == nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{ "error": "missing required key 'enabled'" }`))
			return
		}
		if *bodyStruct.Enabled {
			h.LoggingState.Enable()
		} else {
			h.LoggingState.Disable()
		}
	}

	json.NewEncoder(w).Encode(struct {
		Enabled bool `json:"enabled"`
	}{h.LoggingState.IsEnabled()})
}
