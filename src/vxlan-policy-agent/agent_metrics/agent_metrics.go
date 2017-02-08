package agent_metrics

import "time"

//go:generate counterfeiter -o ../fakes/metrics_sender.go --fake-name MetricsSender . MetricsSender
type MetricsSender interface {
	SendDuration(string, time.Duration)
}

const MetricEnforceDuration = "iptablesEnforceTime"
const MetricPollDuration = "totalPollTime"
const MetricContainerMetadata = "containerMetadataTime"
const MetricPolicyServerPoll = "policyServerPollTime"
