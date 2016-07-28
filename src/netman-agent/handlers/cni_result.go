package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"netman-agent/models"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/store_writer.go --fake-name StoreWriter . storeWriter
type storeWriter interface {
	Add(containerID, groupID, IP string) error
	Del(containerID string) error
}

type CNIResult struct {
	Logger      lager.Logger
	StoreWriter storeWriter
}

func (h *CNIResult) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		h.Logger.Error("body-read", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "body read failed"}`))
		return
	}
	defer req.Body.Close() // not tested

	switch req.Method {
	case "POST":
		var payload models.CNIAddResult
		err = json.Unmarshal(bodyBytes, &payload)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "cannot unmarshal request body as JSON"}`))
			return
		}

		h.Logger.Info("cni_result_add", lager.Data{
			"result": payload,
		})
		err := h.StoreWriter.Add(payload.ContainerID, payload.GroupID, payload.IP)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.Logger.Error("store-add", err)
		}
	case "DELETE":
		var payload models.CNIDelResult
		err = json.Unmarshal(bodyBytes, &payload)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "cannot unmarshal request body as JSON"}`))
			return
		}

		h.Logger.Info("cni_result_del", lager.Data{
			"result": payload,
		})
		err := h.StoreWriter.Del(payload.ContainerID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.Logger.Error("store-del", err)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	w.Write([]byte(`{}`))
}
