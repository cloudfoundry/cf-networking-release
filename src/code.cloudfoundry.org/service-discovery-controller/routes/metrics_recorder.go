package routes

import "sync"

type MetricsRecorder struct {
	requestCount int
	mutex        sync.RWMutex
}

func (m *MetricsRecorder) RecordRequest() {
	m.mutex.Lock()
	m.requestCount += 1
	m.mutex.Unlock()
}

func (m *MetricsRecorder) Getter() (float64, error) {
	m.mutex.Lock()
	count := m.requestCount
	m.requestCount = 0
	m.mutex.Unlock()

	return float64(count), nil
}
