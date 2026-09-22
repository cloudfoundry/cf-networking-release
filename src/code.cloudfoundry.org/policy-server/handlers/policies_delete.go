package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/policy-server/api"
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

// @Summary Delete policies
// @Description Deletes existing network policies. This removes the network connectivity rules between the specified source and destination applications.
// @Tags policies
// @Accept json
// @Produce json
// @Param policies body PoliciesPayload true "Policies to delete"
// @Success 200 {object} object{} "Policies deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or validation failed"
// @Failure 403 {object} ErrorResponse "Forbidden - insufficient permissions"
// @Failure 406 {object} ErrorResponse "Unsupported API version"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /policies/delete [post]
// @Security OAuth2Application[network.write]
func (h *PoliciesDelete) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("delete-policies")
	tokenData := getTokenData(req)

	bodyBytes, err := io.ReadAll(req.Body)
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
	// #nosec G104 - ignore errors writing http responses to avoid spamming logs during a DoS
	w.Write([]byte(`{}`))
}
