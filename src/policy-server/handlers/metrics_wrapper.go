package handlers

import (
	"net/http"
	"time"
)

type MetricWrapper struct {
	Name           string
	MetricsEmitter metricsEmitter
}

func (mw *MetricWrapper) Wrap(handle http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		startTime := time.Now()
		handle.ServeHTTP(w, req)
		mw.MetricsEmitter.EmitAll(map[string]time.Duration{
			mw.Name: time.Now().Sub(startTime),
		})
	})
}
