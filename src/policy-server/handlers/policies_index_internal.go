package handlers

import (
	"lib/marshal"
	"net/http"
	"policy-server/models"
	"policy-server/server_metrics"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/store.go --fake-name Store . store
type store interface {
	All() ([]models.Policy, error)
	Create([]models.Policy) error
	Delete([]models.Policy) error
	Tags() ([]models.Tag, error)
}

//go:generate counterfeiter -o fakes/time_metrics_emitter.go --fake-name MetricsEmitter . metricsEmitter
type metricsEmitter interface {
	EmitDuration(string, time.Duration)
}

type PoliciesIndexInternal struct {
	Logger         lager.Logger
	Store          store
	Marshaler      marshal.Marshaler
	MetricsEmitter metricsEmitter
}

func (h *PoliciesIndexInternal) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	h.Logger.Debug("internal request made to list policies", lager.Data{"URL": req.URL, "RemoteAddr": req.RemoteAddr})
	policies, err := h.Store.All()

	queryDuration := time.Now().Sub(startTime)
	h.MetricsEmitter.EmitDuration(server_metrics.MetricInternalPoliciesQueryDuration, queryDuration)

	if err != nil {
		h.Logger.Error("store-list-policies-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database read failed"}`))
		return
	}

	queryValues := req.URL.Query()
	idList, ok := queryValues["id"]
	if ok {
		ids := strings.Split(idList[0], ",")
		policies = filterByID(policies, ids)
	}

	policyResponse := struct {
		Policies []models.Policy `json:"policies"`
	}{policies}
	bytes, err := h.Marshaler.Marshal(policyResponse)
	if err != nil {
		h.Logger.Error("marshal-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "database marshaling failed"}`))
		return
	}

	w.Write(bytes)
}
