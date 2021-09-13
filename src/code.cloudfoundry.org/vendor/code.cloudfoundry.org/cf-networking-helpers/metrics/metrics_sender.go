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
	ms.SendValue(name, duration.Seconds()*1000, "ms")
}

func (ms *MetricsSender) SendValue(name string, value float64, units string) {
	err := dropsondemetrics.SendValue(name, value, units)
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
