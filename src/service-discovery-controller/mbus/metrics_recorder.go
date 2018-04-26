package mbus

import (
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
)

type MetricsRecorder struct {
	sync.RWMutex
	currentMax time.Duration
	Clock      clock.Clock
}

func NewMetricsRecorder(clock clock.Clock) *MetricsRecorder {
	return &MetricsRecorder{
		Clock: clock,
	}
}

func (r *MetricsRecorder) GetMaxSinceLastInterval() (float64, error) {
	r.Lock()
	duration := r.currentMax
	r.currentMax = 0
	r.Unlock()
	return float64(duration.Nanoseconds()) / float64(time.Millisecond), nil
}

func (r *MetricsRecorder) RecordMessageTransitTime(unixTimeNS int64) {
	if unixTimeNS == 0 {
		return
	}

	r.Lock()
	diff := r.Clock.Now().Sub(time.Unix(0, unixTimeNS))
	if diff > r.currentMax {
		r.currentMax = diff
	}
	r.Unlock()
}
