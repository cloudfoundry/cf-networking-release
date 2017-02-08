package handlers

import (
	"net/http"
	"time"
)

type MetricWrapper struct {
	Name          string
	MetricsSender metricsSender
}

func (mw *MetricWrapper) Wrap(handle http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		startTime := time.Now()
		handle.ServeHTTP(w, req)
		mw.MetricsSender.SendDuration(mw.Name, time.Now().Sub(startTime))
	})
}
