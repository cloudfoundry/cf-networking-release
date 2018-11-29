package store

import (
	"time"
)

//go:generate counterfeiter -o fakes/egress_policy_store.go --fake-name EgressPolicyStore . egressPolicyStore
type egressPolicyStore interface {
	Create([]EgressPolicy) ([]EgressPolicy, error)
	Delete(guids ...string) ([]EgressPolicy, error)
	All() ([]EgressPolicy, error)
	GetBySourceGuids(srcGuids []string) ([]EgressPolicy, error)
	GetByFilter(sourceIds, sourceTypes, destinationIds, destinationNames, appLifecycles []string) ([]EgressPolicy, error)
}

type EgressPolicyMetricsWrapper struct {
	Store         egressPolicyStore
	MetricsSender metricsSender
}

func (mw *EgressPolicyMetricsWrapper) Create(egressPolicies []EgressPolicy) ([]EgressPolicy, error) {
	startTime := time.Now()
	policies, err := mw.Store.Create(egressPolicies)
	createTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("EgressPolicyStoreCreateError")
		mw.MetricsSender.SendDuration("EgressPolicyStoreCreateErrorTime", createTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("EgressPolicyStoreCreateSuccessTime", createTimeDuration)
	}
	return policies, err
}

func (mw *EgressPolicyMetricsWrapper) All() ([]EgressPolicy, error) {
	startTime := time.Now()
	policies, err := mw.Store.All()
	allTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("EgressPolicyStoreAllError")
		mw.MetricsSender.SendDuration("EgressPolicyStoreAllErrorTime", allTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("EgressPolicyStoreAllSuccessTime", allTimeDuration)
	}
	return policies, err
}

func (mw *EgressPolicyMetricsWrapper) Delete(guids ...string) ([]EgressPolicy, error) {
	startTime := time.Now()
	egressPolicies, err := mw.Store.Delete(guids...)
	deleteTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("EgressPolicyStoreDeleteError")
		mw.MetricsSender.SendDuration("EgressPolicyStoreDeleteErrorTime", deleteTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("EgressPolicyStoreDeleteSuccessTime", deleteTimeDuration)
	}
	return egressPolicies, err
}

func (mw *EgressPolicyMetricsWrapper) GetBySourceGuids(srcGuids []string) ([]EgressPolicy, error) {
	startTime := time.Now()
	egressPolicies, err := mw.Store.GetBySourceGuids(srcGuids)
	byGuidsTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("EgressPolicyStoreGetBySourceGuidsError")
		mw.MetricsSender.SendDuration("EgressPolicyStoreGetBySourceGuidsErrorTime", byGuidsTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("EgressPolicyStoreGetBySourceGuidsSuccessTime", byGuidsTimeDuration)
	}
	return egressPolicies, err
}

func (mw *EgressPolicyMetricsWrapper) GetByFilter(sourceIds, sourceTypes, destinationIds, destinationNames, appLifecycles []string) ([]EgressPolicy, error) {
	startTime := time.Now()
	egressPolicies, err := mw.Store.GetByFilter(sourceIds, sourceTypes, destinationIds, destinationNames, appLifecycles)
	byGuidsTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("EgressPolicyStoreGetByFilterError")
		mw.MetricsSender.SendDuration("EgressPolicyStoreGetByFilterErrorTime", byGuidsTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("EgressPolicyStoreGetByFilterSuccessTime", byGuidsTimeDuration)
	}
	return egressPolicies, err
}
