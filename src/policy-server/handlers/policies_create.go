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
	CheckAccess(policyCollection store.PolicyCollection, tokenData uaa_client.CheckTokenResponse) (bool, error)
	CheckEgressPolicyListAccess(userToken uaa_client.CheckTokenResponse) bool
}

//go:generate counterfeiter -o fakes/quota_guard.go --fake-name QuotaGuard . quotaGuard
type quotaGuard interface {
	CheckAccess(policyCollection store.PolicyCollection, tokenData uaa_client.CheckTokenResponse) (bool, error)
}

//go:generate counterfeiter -o fakes/policy_collection_store.go --fake-name PolicyCollectionStore . policyCollectionStore
type policyCollectionStore interface {
	Create(store.PolicyCollection) error
	Delete(store.PolicyCollection) error
}

type PoliciesCreate struct {
	Store         policyCollectionStore
	Mapper        api.PolicyMapper
	PolicyGuard   policyGuard
	QuotaGuard    quotaGuard
	ErrorResponse errorResponse
}

func NewPoliciesCreate(store policyCollectionStore, mapper api.PolicyMapper,
	policyGuard policyGuard, quotaGuard quotaGuard, errorResponse errorResponse) *PoliciesCreate {
	return &PoliciesCreate{
		Store:         store,
		Mapper:        mapper,
		PolicyGuard:   policyGuard,
		QuotaGuard:    quotaGuard,
		ErrorResponse: errorResponse,
	}
}

func (h *PoliciesCreate) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("create-policies")
	tokenData := getTokenData(req)

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		h.ErrorResponse.BadRequest(logger, w, err, "failed reading request body")
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

	authorized, err = h.QuotaGuard.CheckAccess(policies, tokenData)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "check quota failed")
		return
	}
	if !authorized {
		err := errors.New("policy quota exceeded")
		h.ErrorResponse.Forbidden(logger, w, err, err.Error())
		return
	}

	err = h.Store.Create(policies)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "database create failed")
		return
	}

	logger.Info("created-policies", lager.Data{"policies": policies.Policies, "userName": tokenData.UserName})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}
