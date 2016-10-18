package agent_metrics

import (
	"lib/metrics"
	"time"
)

//go:generate counterfeiter -o ../fakes/timer.go --fake-name Timer . timer
type timer interface {
	ElapsedTime(start, end int64) (float64, error)
}

func NewElapsedTimeMetricSource(t timer, name string) metrics.MetricSource {
	start := time.Now().UnixNano()
	elapsedTime := func() (float64, error) {
		end := time.Now().UnixNano()
		return t.ElapsedTime(start, end)
	}
	return metrics.MetricSource{
		Name:   name,
		Unit:   "ms",
		Getter: elapsedTime,
	}
}

type Timer struct{}

func (t Timer) ElapsedTime(start, end int64) (float64, error) {
	return float64(end-start) / 1e6, nil
}
