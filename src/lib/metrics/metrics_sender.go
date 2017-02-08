package metrics

import (
	"time"

	"code.cloudfoundry.org/lager"

	dropsondemetrics "github.com/cloudfoundry/dropsonde/metrics"
)

type MetricsSender struct {
	Logger lager.Logger
}

func (ms *MetricsSender) SendDuration(name string, duration time.Duration) {
	err := dropsondemetrics.SendValue(name, duration.Seconds()*1000, "ms")
	if err != nil {
		ms.Logger.Error("sending-metric", err)
	}
}

func (ms *MetricsSender) IncrementCounter(name string) {
	err := dropsondemetrics.IncrementCounter(name)
	if err != nil {
		ms.Logger.Error("sending-metric", err)
	}
}
