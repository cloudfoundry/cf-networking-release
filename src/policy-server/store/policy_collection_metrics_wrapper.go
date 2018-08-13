package store

import "time"

//go:generate counterfeiter -o fakes/policy_collection_store.go  --fake-name PolicyCollectionStore . policyCollectionStore
type policyCollectionStore interface {
	Create(policyCollection PolicyCollection) error
	Delete(policyCollection PolicyCollection) error
	All() (PolicyCollection, error)
}

type PolicyCollectionMetricsWrapper struct {
	Store         policyCollectionStore
	MetricsSender metricsSender
}

func (p *PolicyCollectionMetricsWrapper) Create(policyCollection PolicyCollection) error {
	startTime := time.Now()
	err := p.Store.Create(policyCollection)
	createDuration := time.Now().Sub(startTime)
	if err != nil {
		p.MetricsSender.IncrementCounter("StoreCreateError")
		p.MetricsSender.SendDuration("StoreCreateErrorTime", createDuration)
	} else {
		p.MetricsSender.SendDuration("StoreCreateSuccessTime", createDuration)
	}
	return err
}

func (p *PolicyCollectionMetricsWrapper) Delete(policyCollection PolicyCollection) error {
	startTime := time.Now()
	err := p.Store.Delete(policyCollection)
	createDuration := time.Now().Sub(startTime)
	if err != nil {
		p.MetricsSender.IncrementCounter("StoreDeleteError")
		p.MetricsSender.SendDuration("StoreDeleteErrorTime", createDuration)
	} else {
		p.MetricsSender.SendDuration("StoreDeleteSuccessTime", createDuration)
	}
	return err
}

func (p *PolicyCollectionMetricsWrapper) All() (PolicyCollection, error) {
	startTime := time.Now()
	collection, err := p.Store.All()
	allDuration := time.Now().Sub(startTime)
	if err != nil {
		p.MetricsSender.IncrementCounter("StoreAllError")
		p.MetricsSender.SendDuration("StoreAllErrorTime", allDuration)
	} else {
		p.MetricsSender.SendDuration("StoreAllSuccessTime", allDuration)
	}
	return collection, err
}
