package store

import (
	"time"
)

//counterfeiter:generate -o fakes/metrics_sender.go --fake-name MetricsSender . metricsSender
type metricsSender interface {
	IncrementCounter(string)
	SendDuration(string, time.Duration)
}

type MetricsWrapper struct {
	Store         Store
	TagStore      TagStore
	MetricsSender metricsSender
}

func (mw *MetricsWrapper) Create(policies []Policy) error {
	startTime := time.Now()
	err := mw.Store.Create(policies)
	createTimeDuration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("StoreCreateError")
		mw.MetricsSender.SendDuration("StoreCreateErrorTime", createTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("StoreCreateSuccessTime", createTimeDuration)
	}
	return err
}

func (mw *MetricsWrapper) All() ([]Policy, error) {
	startTime := time.Now()
	policies, err := mw.Store.All()
	allTimeDuration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("StoreAllError")
		mw.MetricsSender.SendDuration("StoreAllErrorTime", allTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("StoreAllSuccessTime", allTimeDuration)
	}
	return policies, err
}

func (mw *MetricsWrapper) Delete(policies []Policy) error {
	startTime := time.Now()
	err := mw.Store.Delete(policies)
	deleteTimeDuration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("StoreDeleteError")
		mw.MetricsSender.SendDuration("StoreDeleteErrorTime", deleteTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("StoreDeleteSuccessTime", deleteTimeDuration)
	}
	return err
}

func (mw *MetricsWrapper) LastUpdated() (int, error) {
	startTime := time.Now()
	timestamp, err := mw.Store.LastUpdated()
	lastUpdatedTimeDuration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("StoreLastUpdatedError")
		mw.MetricsSender.SendDuration("StoreLastUpdatedErrorTime", lastUpdatedTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("StoreLastUpdatedSuccessTime", lastUpdatedTimeDuration)
	}
	return timestamp, err
}

func (mw *MetricsWrapper) Tags() ([]Tag, error) {
	startTime := time.Now()
	tags, err := mw.TagStore.Tags()
	tagsTimeDuration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("StoreTagsError")
		mw.MetricsSender.SendDuration("StoreTagsErrorTime", tagsTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("StoreTagsSuccessTime", tagsTimeDuration)
	}
	return tags, err
}

func (mw *MetricsWrapper) CreateTag(groupGuid, groupType string) (Tag, error) {
	startTime := time.Now()
	tag, err := mw.TagStore.CreateTag(groupGuid, groupType)
	tagsTimeDuration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("StoreCreateTagError")
		mw.MetricsSender.SendDuration("StoreCreateTagErrorTime", tagsTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("StoreCreateTagSuccessTime", tagsTimeDuration)
	}
	return tag, err
}

func (mw *MetricsWrapper) ByGuids(srcGuids, dstGuids []string, inSourceAndDest bool) ([]Policy, error) {
	startTime := time.Now()
	policies, err := mw.Store.ByGuids(srcGuids, dstGuids, inSourceAndDest)
	byGuidsTimeDuration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("StoreByGuidsError")
		mw.MetricsSender.SendDuration("StoreByGuidsErrorTime", byGuidsTimeDuration)
	} else {
		mw.MetricsSender.SendDuration("StoreByGuidsSuccessTime", byGuidsTimeDuration)
	}
	return policies, err
}

func (mw *MetricsWrapper) CheckDatabase() error {
	startTime := time.Now()
	err := mw.Store.CheckDatabase()
	duration := time.Since(startTime)
	if err != nil {
		mw.MetricsSender.IncrementCounter("StoreCheckDatabaseError")
		mw.MetricsSender.SendDuration("StoreCheckDatabaseErrorTime", duration)
	} else {
		mw.MetricsSender.SendDuration("StoreCheckDatabaseSuccessTime", duration)
	}
	return err
}
