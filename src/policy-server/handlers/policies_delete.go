package handlers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"policy-server/api"
	"policy-server/uaa_client"

	"code.cloudfoundry.org/lager"
)

type PoliciesDelete struct {
	Store         dataStore
	Mapper        api.PolicyMapper
	Validator     validator
	PolicyGuard   policyGuard
	ErrorResponse errorResponse
}

func NewPoliciesDelete(store dataStore, mapper api.PolicyMapper, validator validator,
	policyGuard policyGuard, errorResponse errorResponse) *PoliciesDelete {
	return &PoliciesDelete{
		Store:         store,
		Mapper:        mapper,
		Validator:     validator,
		PolicyGuard:   policyGuard,
		ErrorResponse: errorResponse,
	}
}

func (h *PoliciesDelete) ServeHTTP(logger lager.Logger, w http.ResponseWriter, req *http.Request, tokenData uaa_client.CheckTokenResponse) {
	logger = logger.Session("delete-policies")
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error("failed-reading-request-body", err)
		h.ErrorResponse.BadRequest(w, err, "delete-policies", "invalid request body")
		return
	}

	policies, err := h.Mapper.AsStorePolicy(bodyBytes)
	if err != nil {
		logger.Error("failed-mapping-policies", err)
		h.ErrorResponse.BadRequest(w, err, "delete-policies", fmt.Sprintf("could not map request to store policies: %s", err))
		return
	}

	err = h.Validator.ValidatePolicies(policies)
	if err != nil {
		logger.Error("failed-validating-policies", err)
		h.ErrorResponse.BadRequest(w, err, "delete-policies", err.Error())
		return
	}

	authorized, err := h.PolicyGuard.CheckAccess(policies, tokenData)
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

	err = h.Store.Delete(policies)
	if err != nil {
		logger.Error("failed-deleting-in-database", err)
		h.ErrorResponse.InternalServerError(w, err, "delete-policies", "database delete failed")
		return
	}

	logger.Info("deleted-policies", lager.Data{"policies": policies, "userName": tokenData.UserName})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{}`))
	return
}
