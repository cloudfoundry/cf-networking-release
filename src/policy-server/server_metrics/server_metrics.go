package server_metrics

import (
	"lib/metrics"
	"policy-server/models"
)

const MetricExternalCleanupDuration = "ExternalPoliciesCleanupRequestTime"
const MetricExternalCleanupError = "ExternalPoliciesCleanupError"

const MetricExternalCreateDuration = "ExternalPoliciesCreateRequestTime"
const MetricExternalCreateError = "ExternalPoliciesCreateError"

const MetricExternalDeleteDuration = "ExternalPoliciesDeleteRequestTime"
const MetricExternalDeleteError = "ExternalPoliciesDeleteError"

const MetricExternalIndexDuration = "ExternalPoliciesIndexRequestTime"
const MetricExternalIndexError = "ExternalPoliciesIndexError"

const MetricExternalTagsIndexDuration = "ExternalPoliciesTagsIndexRequestTime"
const MetricExternalTagsIndexError = "ExternalPoliciesTagsIndexError"

const MetricExternalUptimeDuration = "ExternalPoliciesUptimeRequestTime"

const MetricExternalWhoAmIDuration = "ExternalPoliciesWhoAmIRequestTime"
const MetricExternalWhoAmIError = "ExternalPoliciesWhoAmIError"

const MetricInternalPoliciesRequestDuration = "InternalPoliciesRequestTime"
const MetricInternalPoliciesError = "InternalPoliciesError"

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
