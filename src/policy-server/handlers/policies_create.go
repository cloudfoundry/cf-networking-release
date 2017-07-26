package handlers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"policy-server/api"
	"policy-server/store"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/policy_guard.go --fake-name PolicyGuard . policyGuard
type policyGuard interface {
	CheckAccess(policies []store.Policy, tokenData uaa_client.CheckTokenResponse) (bool, error)
}

//go:generate counterfeiter -o fakes/quota_guard.go --fake-name QuotaGuard . quotaGuard
type quotaGuard interface {
	CheckAccess(policies []store.Policy, tokenData uaa_client.CheckTokenResponse) (bool, error)
}

type PoliciesCreate struct {
	Store         dataStore
	Mapper        api.PolicyMapper
	Validator     validator
	PolicyGuard   policyGuard
	QuotaGuard    quotaGuard
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

	policies, err := h.Mapper.AsStorePolicy(bodyBytes)
	if err != nil {
		logger.Error("failed-mapping-policies", err)
		h.ErrorResponse.BadRequest(w, err, "policies-create", fmt.Sprintf("could not map request to store policies: %s", err))
		return
	}

	err = h.Validator.ValidatePolicies(policies)
	if err != nil {
		logger.Error("failed-validating-policies", err)
		h.ErrorResponse.BadRequest(w, err, "policies-create", err.Error())
		return
	}

	authorized, err := h.PolicyGuard.CheckAccess(policies, tokenData)
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

	authorized, err = h.QuotaGuard.CheckAccess(policies, tokenData)
	if err != nil {
		logger.Error("failed-checking-quota", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-create", "check quota failed")
		return
	}
	if !authorized {
		err := errors.New("policy quota exceeded")
		logger.Error("quota-exceeded", err)
		h.ErrorResponse.Forbidden(w, err, "policies-create", err.Error())
		return
	}

	err = h.Store.Create(policies)
	if err != nil {
		logger.Error("failed-creating-in-database", err)
		h.ErrorResponse.InternalServerError(w, err, "policies-create", "database create failed")
		return
	}

	logger.Info("created-policies", lager.Data{"policies": policies, "userName": tokenData.UserName})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
