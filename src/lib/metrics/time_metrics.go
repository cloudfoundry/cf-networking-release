package metrics

import (
	"time"

	"code.cloudfoundry.org/lager"

	dropsondemetrics "github.com/cloudfoundry/dropsonde/metrics"
)

type TimeMetrics struct {
	Logger lager.Logger
}

func (e *TimeMetrics) EmitDuration(name string, duration time.Duration) {
	err := dropsondemetrics.SendValue(name, duration.Seconds()*1000, "ms")
	if err != nil {
		e.Logger.Error("sending-metric", err)
	}
}
