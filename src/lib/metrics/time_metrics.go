package metrics

import (
	"time"

	"code.cloudfoundry.org/lager"

	dropsondemetrics "github.com/cloudfoundry/dropsonde/metrics"
)

type MetricsSender struct {
	Logger lager.Logger
}

func (e *MetricsSender) SendDuration(name string, duration time.Duration) {
	err := dropsondemetrics.SendValue(name, duration.Seconds()*1000, "ms")
	if err != nil {
		e.Logger.Error("sending-metric", err)
	}
}
