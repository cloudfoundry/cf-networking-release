package store

import (
	"policy-server/models"
	"time"
)

//go:generate counterfeiter -o fakes/store.go --fake-name Store . Store
type Store interface {
	Create([]models.Policy) error
	All() ([]models.Policy, error)
	Delete([]models.Policy) error
	Tags() ([]models.Tag, error)
}

//go:generate counterfeiter -o fakes/time_metrics_emitter.go --fake-name MetricsEmitter . metricsEmitter
type metricsEmitter interface {
	EmitAll(map[string]time.Duration)
}

type MetricsWrapper struct {
	Store          Store
	MetricsEmitter metricsEmitter
}

func (mw *MetricsWrapper) Create(policies []models.Policy) error {
	startTime := time.Now()
	err := mw.Store.Create(policies)
	createTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsEmitter.EmitAll(map[string]time.Duration{
			"StoreCreateTime":      createTimeDuration,
			"StoreCreateErrorTime": createTimeDuration,
		})
	} else {
		mw.MetricsEmitter.EmitAll(map[string]time.Duration{
			"StoreCreateTime": createTimeDuration,
		})
	}
	return err
}

func (mw *MetricsWrapper) All() ([]models.Policy, error) {
	startTime := time.Now()
	policies, err := mw.Store.All()
	allTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsEmitter.EmitAll(map[string]time.Duration{
			"StoreAllTime":      allTimeDuration,
			"StoreAllErrorTime": allTimeDuration,
		})
	} else {
		mw.MetricsEmitter.EmitAll(map[string]time.Duration{
			"StoreAllTime": allTimeDuration,
		})
	}
	return policies, err
}

func (mw *MetricsWrapper) Delete(policies []models.Policy) error {
	startTime := time.Now()
	err := mw.Store.Delete(policies)
	deleteTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsEmitter.EmitAll(map[string]time.Duration{
			"StoreDeleteTime":      deleteTimeDuration,
			"StoreDeleteErrorTime": deleteTimeDuration,
		})
	} else {
		mw.MetricsEmitter.EmitAll(map[string]time.Duration{
			"StoreDeleteTime": deleteTimeDuration,
		})
	}
	return err
}

func (mw *MetricsWrapper) Tags() ([]models.Tag, error) {
	startTime := time.Now()
	tags, err := mw.Store.Tags()
	allTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsEmitter.EmitAll(map[string]time.Duration{
			"StoreTagsTime":      allTimeDuration,
			"StoreTagsErrorTime": allTimeDuration,
		})
	} else {
		mw.MetricsEmitter.EmitAll(map[string]time.Duration{
			"StoreTagsTime": allTimeDuration,
		})
	}
	return tags, err
}
