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

//go:generate counterfeiter -o fakes/policy_guard.go --fake-name PolicyGuard . policyGuard
type policyGuard interface {
	CheckAccess(policies []models.Policy, tokenData uaa_client.CheckTokenResponse) (bool, error)
}

//go:generate counterfeiter -o fakes/quota_guard.go --fake-name QuotaGuard . quotaGuard
type quotaGuard interface {
	CheckAccess(policies []models.Policy, tokenData uaa_client.CheckTokenResponse) (bool, error)
}

type PoliciesCreate struct {
	Store         store
	Unmarshaler   marshal.Unmarshaler
	Validator     validator
	PolicyGuard   policyGuard
	ErrorResponse errorResponse
}

func (h *PoliciesCreate) ServeHTTP(logger lager.Logger, w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
	logger = logger.Session("create-policies")
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error("failed-reading-request-body", err)
		h.ErrorResponse.BadRequest(w, err, "policies-create", "failed reading request body")
		return
	}

	var payload struct {
		Policies []models.Policy `json:"policies"`
	}
	err = h.Unmarshaler.Unmarshal(bodyBytes, &payload)
	if err != nil {
		logger.Error("failed-unmarshalling-payload", err)
		h.ErrorResponse.BadRequest(w, err, "policies-create", "invalid values passed to API")
		return
	}

	err = h.Validator.ValidatePolicies(payload.Policies)
	if err != nil {
		logger.Error("failed-validating-policies", err)
		h.ErrorResponse.BadRequest(w, err, "policies-create", err.Error())
		return
	}

	authorized, err := h.PolicyGuard.CheckAccess(payload.Policies, tokenData)
	if err != nil {
		logger.Error("failed-checking-access", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-create", "check access failed")
		return
	}
	if !authorized {
		err := errors.New("one or more applications cannot be found or accessed")
		logger.Error("failed-authorizing", err)
		h.ErrorResponse.Forbidden(w, err, "policies-create", err.Error())
		return
	}

	err = h.Store.Create(payload.Policies)
	if err != nil {
		logger.Error("failed-creating-in-database", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-create", "database create failed")
		return
	}

	logger.Info("created-policies", lager.Data{"policies": payload.Policies, "userName": tokenData.UserName})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
