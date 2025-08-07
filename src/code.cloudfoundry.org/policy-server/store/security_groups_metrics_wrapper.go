package store

import (
	"time"
)

type securityGroupsStore interface {
	Replace([]SecurityGroup) error
	BySpaceGuids([]string, Page) ([]SecurityGroup, Pagination, error)
	UpdateSpaceCache(SpaceCache) error
	LastUpdated() (int, error)
	CheckForASGUpdates([]string, time.Time) (bool, error)
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

func (mw *SecurityGroupsMetricsWrapper) CheckForASGUpdates(spaceGuids []string, since time.Time) (bool, error) {
	startTime := time.Now()
	updates, err := mw.Store.CheckForASGUpdates(spaceGuids, since)
	allTimeDuration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("SecurityGroupsStoreCheckForASGUpdatesError")
		mw.MetricsSender.SendDuration("SecurityGroupsStoreCheckForASGUpdatesErrorTime", allTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("SecurityGroupsStoreCheckForASGUpdatesSuccessTime", allTimeDuration)
	}
	return updates, err
}

func (mw *SecurityGroupsMetricsWrapper) UpdateSpaceCache(spaceCache SpaceCache) error {
	startTime := time.Now()
	err := mw.Store.UpdateSpaceCache(spaceCache)
	allTimeDuration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("SecurityGroupsUpdateSpaceCacheStoreError")
		mw.MetricsSender.SendDuration("SecurityGroupsStoreUpdateSpaceCacheErrorTime", allTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("SecurityGroupsStoreUpdateSpaceCacheSuccessTime", allTimeDuration)
	}
	return err
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
