package handlers

import (
	"fmt"
	"io/ioutil"
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/store"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager"
)

type PoliciesDelete struct {
	Logger      lager.Logger
	Unmarshaler marshal.Unmarshaler
	Store       store.Store
	Validator   validator
}

func (h *PoliciesDelete) ServeHTTP(w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
	if req.Method != "POST" {
		h.Logger.Info("policy-delete", lager.Data{"warning": "legacy-mechanism", "method": req.Method})
	}

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		h.Logger.Error("body-read-failed", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request body format passed to API should be JSON"}`))
		return
	}

	var payload struct {
		Policies []models.Policy `json:"policies"`
	}
	err = h.Unmarshaler.Unmarshal(bodyBytes, &payload)
	if err != nil {
		h.Logger.Error("unmarshal-failed", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid values passed to API"}`))
		return
	}

	if err = h.Validator.ValidatePolicies(payload.Policies); err != nil {
		h.Logger.Error("bad-request", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err)))
		return
	}

	err = h.Store.Delete(payload.Policies)
	if err != nil {
		h.Logger.Error("store-delete-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database delete failed"}`))
		return
	}

	h.Logger.Info("policy-delete", lager.Data{"policies": payload.Policies, "userName": tokenData.UserName})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{}`))
	return
}
