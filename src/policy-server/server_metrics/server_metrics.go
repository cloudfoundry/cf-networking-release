package server_metrics

import (
	"lib/metrics"
	"policy-server/models"
)

const MetricInternalPoliciesRequestDuration = "InternalPoliciesRequestTime"
const MetricInternalPoliciesQueryDuration = "InternalPoliciesQueryTime"

//go:generate counterfeiter -o fakes/store.go --fake-name Store . store
type store interface {
	All() ([]models.Policy, error)
}

func NewTotalPoliciesSource(lister store) metrics.MetricSource {
	return metrics.MetricSource{
		Name: "totalPolicies",
		Unit: "",
		Getter: func() (float64, error) {
			allPolicies, err := lister.All()
			return float64(len(allPolicies)), err
		},
	}
}
