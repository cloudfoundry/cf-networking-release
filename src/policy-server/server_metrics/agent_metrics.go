package server_metrics

import (
	"time"

	"code.cloudfoundry.org/lager"

	dropsondemetrics "github.com/cloudfoundry/dropsonde/metrics"
)

const MetricInternalPoliciesRequestDuration = "InternalPoliciesRequestTime"
const MetricInternalPoliciesQueryDuration = "InternalPoliciesQueryTime"

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
