package store

import (
	"time"
)

type securityGroupsStore interface {
	Replace([]SecurityGroup) error
	BySpaceGuids([]string, Page) ([]SecurityGroup, Pagination, error)
	LastUpdated() (int, error)
}

type SecurityGroupsMetricsWrapper struct {
	Store         securityGroupsStore
	MetricsSender metricsSender
}

func (sw *SecurityGroupsMetricsWrapper) Replace(newSecurityGroups []SecurityGroup) error {
	startTime := time.Now()
	err := sw.Store.Replace(newSecurityGroups)
	createTimeDuration := time.Since(startTime)
	if err != nil {
		sw.MetricsSender.IncrementCounter("SecurityGroupsStoreReplaceError")
		sw.MetricsSender.SendDuration("SecurityGroupsStoreReplaceErrorTime", createTimeDuration)
	} else {
		sw.MetricsSender.SendDuration("SecurityGroupsStoreReplaceSuccessTime", createTimeDuration)
	}
	return err
}

func (mw *SecurityGroupsMetricsWrapper) BySpaceGuids(spaceGuids []string, page Page) ([]SecurityGroup, Pagination, error) {
	startTime := time.Now()
	securityGroups, pagination, err := mw.Store.BySpaceGuids(spaceGuids, page)
	allTimeDuration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("SecurityGroupsStoreBySpaceGuidsError")
		mw.MetricsSender.SendDuration("SecurityGroupsStoreBySpaceGuidsErrorTime", allTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("SecurityGroupsStoreBySpaceGuidsSuccessTime", allTimeDuration)
	}
	return securityGroups, pagination, err
}

func (mw *SecurityGroupsMetricsWrapper) LastUpdated() (int, error) {
	startTime := time.Now()
	timestamp, err := mw.Store.LastUpdated()
	lastUpdatedTimeDuration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("SecurityGroupsStoreLastUpdatedError")
		mw.MetricsSender.SendDuration("SecurityGroupsStoreLastUpdatedErrorTime", lastUpdatedTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("SecurityGroupsStoreLastUpdatedSuccessTime", lastUpdatedTimeDuration)
	}
	return timestamp, err
}
