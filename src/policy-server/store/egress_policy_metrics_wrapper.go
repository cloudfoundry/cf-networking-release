package store

import (
	"time"
)

//go:generate counterfeiter -o fakes/egress_policy_store.go --fake-name EgressPolicyStore . egressPolicyStore
type egressPolicyStore interface {
	Create([]EgressPolicy) error
	Delete([]EgressPolicy) error
	All() ([]EgressPolicy, error)
	ByGuids(srcGuids []string) ([]EgressPolicy, error)
}

type EgressPolicyMetricsWrapper struct {
	Store         egressPolicyStore
	MetricsSender metricsSender
}

func (mw *EgressPolicyMetricsWrapper) Create(egressPolicies []EgressPolicy) error {
	startTime := time.Now()
	err := mw.Store.Create(egressPolicies)
	createTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("EgressPolicyStoreCreateError")
		mw.MetricsSender.SendDuration("EgressPolicyStoreCreateErrorTime", createTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("EgressPolicyStoreCreateSuccessTime", createTimeDuration)
	}
	return err
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

func (mw *EgressPolicyMetricsWrapper) Delete(egressPolicies []EgressPolicy) error {
	startTime := time.Now()
	err := mw.Store.Delete(egressPolicies)
	deleteTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("EgressPolicyStoreDeleteError")
		mw.MetricsSender.SendDuration("EgressPolicyStoreDeleteErrorTime", deleteTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("EgressPolicyStoreDeleteSuccessTime", deleteTimeDuration)
	}
	return err
}

func (mw *EgressPolicyMetricsWrapper) ByGuids(srcGuids []string) ([]EgressPolicy, error) {
	startTime := time.Now()
	egressPolicies, err := mw.Store.ByGuids(srcGuids)
	byGuidsTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("EgressPolicyStoreByGuidsError")
		mw.MetricsSender.SendDuration("EgressPolicyStoreByGuidsErrorTime", byGuidsTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("EgressPolicyStoreByGuidsSuccessTime", byGuidsTimeDuration)
	}
	return egressPolicies, err
}
