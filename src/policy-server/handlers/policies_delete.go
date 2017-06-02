package handlers

import (
	"errors"
	"io/ioutil"
	"net/http"
	"policy-server/models"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/lager"
)

type PoliciesDelete struct {
	Unmarshaler   marshal.Unmarshaler
	Store         store
	Validator     validator
	PolicyGuard   policyGuard
	ErrorResponse errorResponse
}

func (h *PoliciesDelete) ServeHTTP(logger lager.Logger, w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
	logger = logger.Session("delete-policies")
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error("failed-reading-request-body", err)
		h.ErrorResponse.BadRequest(w, err, "delete-policies", "invalid request body")
		return
	}

	var payload struct {
		Policies []models.Policy `json:"policies"`
	}
	err = h.Unmarshaler.Unmarshal(bodyBytes, &payload)
	if err != nil {
		logger.Error("failed-unmarshalling-payload", err)
		h.ErrorResponse.BadRequest(w, err, "delete-policies", "invalid values passed to API")
		return
	}

	err = h.Validator.ValidatePolicies(payload.Policies)
	if err != nil {
		logger.Error("failed-validating-policies", err)
		h.ErrorResponse.BadRequest(w, err, "delete-policies", err.Error())
		return
	}

	authorized, err := h.PolicyGuard.CheckAccess(payload.Policies, tokenData)
	if err != nil {
		logger.Error("failed-checking-access", err)
		h.ErrorResponse.InternalServerError(w, err, "delete-policies", "check access failed")
		return
	}
	if !authorized {
		err := errors.New("one or more applications cannot be found or accessed")
		logger.Error("failed-authorizing-access", err)
		h.ErrorResponse.Forbidden(w, err, "delete-policies", err.Error())
		return
	}

	err = h.Store.Delete(payload.Policies)
	if err != nil {
		logger.Error("failed-deleting-in-database", err)
		h.ErrorResponse.InternalServerError(w, err, "delete-policies", "database delete failed")
		return
	}

	logger.Info("deleted-policies", lager.Data{"policies": payload.Policies, "userName": tokenData.UserName})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{}`))
	return
}
