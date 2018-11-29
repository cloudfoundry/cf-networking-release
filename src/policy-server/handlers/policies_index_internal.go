package handlers

import (
	"net/http"
	"net/url"
	"policy-server/api"
	"policy-server/store"
	"strings"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/egress_policy_store.go --fake-name EgressPolicyStore . egressPolicyStore
type egressPolicyStore interface {
	All() ([]store.EgressPolicy, error)
	GetByFilter(sourceId, sourceType, destinationId, destinationName, appLifecycle []string) ([]store.EgressPolicy, error)
	GetBySourceGuids(ids []string) ([]store.EgressPolicy, error)
	Create(egressPolicies []store.EgressPolicy) ([]store.EgressPolicy, error)
	Delete(guids ...string) ([]store.EgressPolicy, error)
}

type PoliciesIndexInternal struct {
	Logger                                   lager.Logger
	Store                                    store.Store
	PolicyCollectionWriter                   api.PolicyCollectionWriter
	ErrorResponse                            errorResponse
	EgressStore                              egressPolicyStore
	EnforceExperimentalDynamicEgressPolicies bool
}

func NewPoliciesIndexInternal(logger lager.Logger, store store.Store, egressStore egressPolicyStore,
	writer api.PolicyCollectionWriter, errorResponse errorResponse, enforceExperimentalDynamicEgressPolicies bool) *PoliciesIndexInternal {
	return &PoliciesIndexInternal{
		Logger:                                   logger,
		Store:                                    store,
		EgressStore:                              egressStore,
		PolicyCollectionWriter:                   writer,
		ErrorResponse:                            errorResponse,
		EnforceExperimentalDynamicEgressPolicies: enforceExperimentalDynamicEgressPolicies,
	}
}

func (h *PoliciesIndexInternal) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logger := getLogger(req)
	logger = logger.Session("index-policies-internal")

	queryValues := req.URL.Query()
	ids := parseIds(queryValues)

	var policies []store.Policy
	var err error
	if len(ids) == 0 {
		policies, err = h.Store.All()
	} else {
		policies, err = h.Store.ByGuids(ids, ids, false)
	}

	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "database read failed")
		return
	}

	var egressPolicies []store.EgressPolicy
	if h.EnforceExperimentalDynamicEgressPolicies {
		if len(ids) == 0 {
			egressPolicies, err = h.EgressStore.All()
		} else {
			egressPolicies, err = h.EgressStore.GetBySourceGuids(ids)
		}
		if err != nil {
			h.ErrorResponse.InternalServerError(logger, w, err, "egress database read failed")
			return
		}
	}

	bytes, err := h.PolicyCollectionWriter.AsBytes(policies, egressPolicies)
	if err != nil {
		h.ErrorResponse.InternalServerError(logger, w, err, "map policies as bytes failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

func parseIds(queryValues url.Values) []string {
	var ids []string
	idList, ok := queryValues["id"]
	if ok {
		ids = strings.Split(idList[0], ",")
	}
	return ids
}
