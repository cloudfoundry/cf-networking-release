package metrics

import (
	"os"
	"time"

	"code.cloudfoundry.org/lager"

	dropsondemetrics "github.com/cloudfoundry/dropsonde/metrics"
)

type MetricSource struct {
	Name   string
	Unit   string
	Getter func() (float64, error)
}

type MetricsEmitter struct {
	logger   lager.Logger
	interval time.Duration
	metrics  []MetricSource
}

func NewMetricsEmitter(logger lager.Logger, interval time.Duration, metrics ...MetricSource) *MetricsEmitter {
	return &MetricsEmitter{
		logger:   logger,
		interval: interval,
		metrics:  metrics,
	}
}

func (m *MetricsEmitter) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	m.emitMetrics()
	close(ready)

	for {
		select {
		case <-signals:
			return nil
		case <-time.After(m.interval):
			m.emitMetrics()
		}
	}
}

func (m *MetricsEmitter) emitMetrics() {
	for _, source := range m.metrics {
		value, err := source.Getter()
		if err != nil {
			m.logger.Error("metric-getter", err, lager.Data{"source": source.Name})
			continue
		}

		dropsondemetrics.SendValue(source.Name, value, source.Unit)
	}
}

func (m *MetricsEmitter) EmitMetrics() {
	m.emitMetrics()
}
