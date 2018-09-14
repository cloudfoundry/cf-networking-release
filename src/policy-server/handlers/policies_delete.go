package handlers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"policy-server/api"

	"code.cloudfoundry.org/lager"
)

type PoliciesDelete struct {
	Store         policyStore
	Mapper        api.PolicyMapper
	PolicyGuard   policyGuard
	ErrorResponse errorResponse
}

func NewPoliciesDelete(store policyStore, mapper api.PolicyMapper,
	policyGuard policyGuard, errorResponse errorResponse) *PoliciesDelete {
	return &PoliciesDelete{
		Store:         store,
		Mapper:        mapper,
		PolicyGuard:   policyGuard,
		ErrorResponse: errorResponse,
	}
}

func (h *PoliciesDelete) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("delete-policies")
	tokenData := getTokenData(req)

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		h.ErrorResponse.BadRequest(logger, w, err, "invalid request body")
		return
	}

	policies, err := h.Mapper.AsStorePolicy(bodyBytes)
	if err != nil {
		h.ErrorResponse.BadRequest(logger, w, err, fmt.Sprintf("mapper: %s", err))
		return
	}

	authorized, err := h.PolicyGuard.CheckAccess(policies, tokenData)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "check access failed")
		return
	}
	if !authorized {
		err := errors.New("one or more applications cannot be found or accessed")
		h.ErrorResponse.Forbidden(logger, w, err, err.Error())
		return
	}

	err = h.Store.Delete(policies)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "database delete failed")
		return
	}

	logger.Info("deleted-policies", lager.Data{"policies": policies, "userName": tokenData.UserName})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{}`))
	return
}
