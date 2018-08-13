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
		p.MetricsSender.IncrementCounter("CollectionStoreCreateError")
		p.MetricsSender.SendDuration("CollectionStoreCreateErrorTime", createDuration)
	} else {
		p.MetricsSender.SendDuration("CollectionStoreCreateSuccessTime", createDuration)
	}
	return err
}

func (p *PolicyCollectionMetricsWrapper) Delete(policyCollection PolicyCollection) error {
	startTime := time.Now()
	err := p.Store.Delete(policyCollection)
	createDuration := time.Now().Sub(startTime)
	if err != nil {
		p.MetricsSender.IncrementCounter("CollectionStoreDeleteError")
		p.MetricsSender.SendDuration("CollectionStoreDeleteErrorTime", createDuration)
	} else {
		p.MetricsSender.SendDuration("CollectionStoreDeleteSuccessTime", createDuration)
	}
	return err
}

func (p *PolicyCollectionMetricsWrapper) All() (PolicyCollection, error) {
	startTime := time.Now()
	collection, err := p.Store.All()
	allDuration := time.Now().Sub(startTime)
	if err != nil {
		p.MetricsSender.IncrementCounter("CollectionStoreAllError")
		p.MetricsSender.SendDuration("CollectionStoreAllErrorTime", allDuration)
	} else {
		p.MetricsSender.SendDuration("CollectionStoreAllSuccessTime", allDuration)
	}
	return collection, err
}
