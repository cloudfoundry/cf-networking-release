package asg_syncer

//go:generate counterfeiter -generate

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/policy-server/cc_client"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/uaa_client"
)

const metricSecurityGroupsRetrievalFromCCDuration = "SecurityGroupsRetrievalFromCCTime"
const metricSecurityGroupsTotalSyncDuration = "SecurityGroupsTotalSyncTime"

//go:generate counterfeiter -o fakes/metrics_sender.go --fake-name MetricsSender . metricsSender
type metricsSender interface {
	SendDuration(string, time.Duration)
}

type ASGSyncer struct {
	Logger           lager.Logger
	Store            store.SecurityGroupsStore
	UAAClient        uaa_client.UAAClient
	CCClient         cc_client.CCClient
	PollInterval     time.Duration
	MetricsSender    metricsSender
	RetryDeadline    time.Duration
	latestUpdateTime time.Time
	lastSyncTime     time.Time
	Clock            clock.Clock
}

func NewASGSyncer(logger lager.Logger, store store.SecurityGroupsStore, uaaClient uaa_client.UAAClient, ccClient cc_client.CCClient, pollInterval time.Duration, metricsSender metricsSender, retryDeadline time.Duration) *ASGSyncer {
	return &ASGSyncer{
		Logger:        logger,
		Store:         store,
		UAAClient:     uaaClient,
		CCClient:      ccClient,
		PollInterval:  pollInterval,
		MetricsSender: metricsSender,
		RetryDeadline: retryDeadline,
		Clock:         clock.NewClock(),
	}
}

func (a *ASGSyncer) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)
	for {
		select {
		case <-signals:
			return nil
		case <-time.After(a.PollInterval):
			if err := a.Poll(); err != nil {
				a.Logger.Error("asg-sync-cycle", err)
				return err
			}
		}
	}
}

func valueHasNotBeenUpdated(ccTimetamp, localTimestamp time.Time) bool {
	return !ccTimetamp.IsZero() && !ccTimetamp.After(localTimestamp)
}

func (a *ASGSyncer) Poll() error {
	syncStartTime := a.Clock.Now()
	a.Logger.Debug("asg-sync-started")
	defer a.Logger.Debug("asg-sync-complete")

	if a.lastSyncTime.IsZero() {
		a.Logger.Debug("initializing-lastSyncTime")
		a.lastSyncTime = a.Clock.Now()
	}

	token, err := a.UAAClient.GetToken()
	if err != nil {
		return err
	}

	ccLatestUpdateTime, err := a.CCClient.GetSecurityGroupsLastUpdate(token)
	if err != nil {
		return err
	}

	if valueHasNotBeenUpdated(ccLatestUpdateTime, a.latestUpdateTime) {
		a.Logger.Debug("skipping-update", lager.Data{"cc-latest-update-time": ccLatestUpdateTime, "local-latest-update-time": a.latestUpdateTime})
		return nil
	}

	a.Logger.Debug("performing-update", lager.Data{"cc-latest-update-time": ccLatestUpdateTime, "local-latest-update-time": a.latestUpdateTime})

	retrieveStartTime := a.Clock.Now()
	ccSGs, err := a.CCClient.GetSecurityGroups(token)
	if err != nil {
		if _, ok := err.(cc_client.UnstableSecurityGroupListError); ok {
			if a.Clock.Now().After(a.lastSyncTime.Add(a.RetryDeadline)) {
				return fmt.Errorf("unable to retrieve a consistent listing of security groups from CAPI after '%s': %s", a.RetryDeadline, err)
			}
			return nil
		}
		return err
	}
	a.Logger.Info("successfully-fetched-security-groups", lager.Data{"count": len(ccSGs)})

	retrieveEndTime := a.Clock.Now()
	a.MetricsSender.SendDuration(metricSecurityGroupsRetrievalFromCCDuration, retrieveEndTime.Sub(retrieveStartTime))
	a.Logger.Debug("successfully-sent-performance-metrics")

	a.Logger.Debug("updating-local-latest-update-time", lager.Data{"old-local-latest-update-time": a.latestUpdateTime, "new-local-latest-update-time": ccLatestUpdateTime})
	a.latestUpdateTime = ccLatestUpdateTime

	a.lastSyncTime = a.Clock.Now()
	a.Logger.Debug("updating-last-sync-time", lager.Data{"new-last-sync-time": a.lastSyncTime})

	sgs := []store.SecurityGroup{}
	for _, ccSG := range ccSGs {
		stagingSpaces := []string{}
		for _, data := range ccSG.Relationships.StagingSpaces.Data {
			if guid, ok := data["guid"]; ok {
				stagingSpaces = append(stagingSpaces, guid)
			} else {
				return fmt.Errorf("no 'guid' found for staging-space-relationship on asg '%s'", ccSG.GUID)
			}
		}
		runningSpaces := []string{}
		for _, data := range ccSG.Relationships.RunningSpaces.Data {
			if guid, ok := data["guid"]; ok {
				runningSpaces = append(runningSpaces, guid)
			} else {
				return fmt.Errorf("no 'guid' found for running-space-relationship on asg '%s'", ccSG.GUID)
			}
		}
		rules, err := json.Marshal(ccSG.Rules)
		if err != nil {
			return fmt.Errorf("error converting rules to json for ASG '%s': %s", ccSG.GUID, err)
		}
		sgs = append(sgs, store.SecurityGroup{
			Guid:              ccSG.GUID,
			Name:              ccSG.Name,
			Rules:             string(rules),
			StagingDefault:    ccSG.GloballyEnabled.Staging,
			RunningDefault:    ccSG.GloballyEnabled.Running,
			StagingSpaceGuids: stagingSpaces,
			RunningSpaceGuids: runningSpaces,
		})
	}

	err = a.Store.Replace(sgs)

	syncEndTime := a.Clock.Now()
	a.MetricsSender.SendDuration(metricSecurityGroupsTotalSyncDuration, syncEndTime.Sub(syncStartTime))

	return err
}
