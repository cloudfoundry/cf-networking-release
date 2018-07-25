package store

import (
	"policy-server/db"
	"time"
)

type EgressPolicyMetricsWrapper struct {
	Store         egressPolicyStore
	MetricsSender metricsSender
}

func (mw *EgressPolicyMetricsWrapper) CreateWithTx(tx db.Transaction, egressPolicies []EgressPolicy) error {
	startTime := time.Now()
	err := mw.Store.CreateWithTx(tx, egressPolicies)
	createTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("EgressPolicyStoreCreateWithTxError")
		mw.MetricsSender.SendDuration("EgressPolicyStoreCreateWithTxErrorTime", createTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("EgressPolicyStoreCreateWithTxSuccessTime", createTimeDuration)
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

func (mw *EgressPolicyMetricsWrapper) DeleteWithTx(tx db.Transaction, egressPolicies []EgressPolicy) error {
	startTime := time.Now()
	err := mw.Store.DeleteWithTx(tx, egressPolicies)
	deleteTimeDuration := time.Now().Sub(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("EgressPolicyStoreDeleteWithTxError")
		mw.MetricsSender.SendDuration("EgressPolicyStoreDeleteWithTxErrorTime", deleteTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("EgressPolicyStoreDeleteWithTxSuccessTime", deleteTimeDuration)
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