package agent_metrics

import (
	"time"

	"code.cloudfoundry.org/lager"

	dropsondemetrics "github.com/cloudfoundry/dropsonde/metrics"
)

//go:generate counterfeiter -o ../fakes/time_metrics_emitter.go --fake-name TimeMetricsEmitter . TimeMetricsEmitter
type TimeMetricsEmitter interface {
	EmitAll(map[string]time.Duration)
}

const MetricEnforceDuration = "iptablesEnforceTime"
const MetricPollDuration = "totalPollTime"
const MetricContainerMetadata = "containerMetadataTime"
const MetricPolicyServerPoll = "policyServerPollTime"

type TimeMetrics struct {
	Logger lager.Logger
}

func (e *TimeMetrics) EmitAll(durations map[string]time.Duration) {
	for name, duration := range durations {
		err := dropsondemetrics.SendValue(name, duration.Seconds()*1000, "ms")
		if err != nil {
			e.Logger.Error("sending-metric", err) // not tested
		}
	}
}
