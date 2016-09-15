package server_metrics

import (
	"lib/metrics"
	"policy-server/models"
)

type allPoliciesGetter interface {
	All() ([]models.Policy, error)
}

func NewTotalPoliciesSource(lister allPoliciesGetter) metrics.MetricSource {
	return metrics.MetricSource{
		Name: "totalPolicies",
		Unit: "",
		Getter: func() (float64, error) {
			allPolicies, err := lister.All()
			return float64(len(allPolicies)), err
		},
	}
}
