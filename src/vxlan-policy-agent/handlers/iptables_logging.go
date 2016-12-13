package handlers

import (
	"encoding/json"
	"net/http"
)

type IPTablesLogging struct {
	enabled     bool
	LoggingChan chan bool
}

func (h *IPTablesLogging) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var bodyStruct = struct {
		Enabled *bool `json:"enabled"`
	}{}

	if r.Method == "PUT" {
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
		h.enabled = *bodyStruct.Enabled
		h.LoggingChan <- h.enabled
		return
	}

	bodyStruct.Enabled = &h.enabled
	json.NewEncoder(w).Encode(bodyStruct)
}
