package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pivotal-golang/lager"
)

type CNIResult struct {
	Logger lager.Logger
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

	var data interface{}
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "cannot unmarshal request body as JSON"}`))
		return
	}

	switch req.Method {
	case "POST":
		h.Logger.Info("cni_result_add", lager.Data{
			"result": data,
		})
	case "DELETE":
		h.Logger.Info("cni_result_del", lager.Data{
			"result": data,
		})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	w.Write([]byte(`{}`))
}
