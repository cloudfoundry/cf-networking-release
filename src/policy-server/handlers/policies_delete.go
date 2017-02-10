package handlers

import (
	"fmt"
	"io/ioutil"
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager"
)

type PoliciesDelete struct {
	Logger        lager.Logger
	Unmarshaler   marshal.Unmarshaler
	Store         store
	Validator     validator
	PolicyGuard   policyGuard
	ErrorResponse errorResponse
}

func (h *PoliciesDelete) ServeHTTP(w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
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

	err = h.Validator.ValidatePolicies(payload.Policies)
	if err != nil {
		h.Logger.Error("bad-request", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err)))
		return
	}

	authorized, err := h.PolicyGuard.CheckAccess(payload.Policies, tokenData)
	if err != nil {
		h.ErrorResponse.InternalServerError(w, err, "policies-delete", "check access failed")
		return
	}
	if !authorized {
		message := "one or more applications cannot be found or accessed"
		h.Logger.Info(fmt.Sprintf("check-access-failed: %s", message))
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, message)))
		return
	}

	err = h.Store.Delete(payload.Policies)
	if err != nil {
		h.ErrorResponse.InternalServerError(w, err, "policies-delete", "database delete failed")
		return
	}

	h.Logger.Info("policy-delete", lager.Data{"policies": payload.Policies, "userName": tokenData.UserName})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{}`))
	return
}
