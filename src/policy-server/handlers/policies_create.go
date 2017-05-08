package handlers

import (
	"errors"
	"io/ioutil"
	"net/http"
	"policy-server/models"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/go-db-helpers/marshal"
	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/policy_guard.go --fake-name PolicyGuard . policyGuard
type policyGuard interface {
	CheckAccess(policies []models.Policy, tokenData uaa_client.CheckTokenResponse) (bool, error)
}

type PoliciesCreate struct {
	Logger        lager.Logger
	Store         store
	Unmarshaler   marshal.Unmarshaler
	Validator     validator
	PolicyGuard   policyGuard
	ErrorResponse errorResponse
}

func (h *PoliciesCreate) ServeHTTP(w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		h.ErrorResponse.BadRequest(w, err, "policies-create", "failed reading request body")
		return
	}

	var payload struct {
		Policies []models.Policy `json:"policies"`
	}
	err = h.Unmarshaler.Unmarshal(bodyBytes, &payload)
	if err != nil {
		h.ErrorResponse.BadRequest(w, err, "policies-create", "invalid values passed to API")
		return
	}

	err = h.Validator.ValidatePolicies(payload.Policies)
	if err != nil {
		h.ErrorResponse.BadRequest(w, err, "policies-create", err.Error())
		return
	}

	authorized, err := h.PolicyGuard.CheckAccess(payload.Policies, tokenData)
	if err != nil {
		h.ErrorResponse.InternalServerError(w, err, "policies-create", "check access failed")
		return
	}
	if !authorized {
		err := errors.New("one or more applications cannot be found or accessed")
		h.ErrorResponse.Forbidden(w, err, "policies-create", err.Error())
		return
	}

	err = h.Store.Create(payload.Policies)
	if err != nil {
		h.ErrorResponse.InternalServerError(w, err, "policies-create", "database create failed")
		return
	}

	h.Logger.Info("policy-create", lager.Data{"policies": payload.Policies, "userName": tokenData.UserName})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
