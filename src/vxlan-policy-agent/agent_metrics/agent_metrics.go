package agent_metrics

import "time"

//go:generate counterfeiter -o ../fakes/time_metrics_emitter.go --fake-name TimeMetricsEmitter . TimeMetricsEmitter
type TimeMetricsEmitter interface {
	EmitAll(map[string]time.Duration)
}

const MetricEnforceDuration = "iptablesEnforceTime"
const MetricPollDuration = "totalPollTime"
const MetricContainerMetadata = "containerMetadataTime"
const MetricPolicyServerPoll = "policyServerPollTime"
