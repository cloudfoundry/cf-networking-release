package asg_syncer

//go:generate counterfeiter -generate

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
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
	Logger        lager.Logger
	Store         store.SecurityGroupsStore
	UAAClient     uaa_client.UAAClient
	CCClient      cc_client.CCClient
	PollInterval  time.Duration
	MetricsSender metricsSender
}

func NewASGSyncer(logger lager.Logger, store store.SecurityGroupsStore, uaaClient uaa_client.UAAClient, ccClient cc_client.CCClient, pollInterval time.Duration, metricsSender metricsSender) *ASGSyncer {
	return &ASGSyncer{
		Logger:        logger,
		Store:         store,
		UAAClient:     uaaClient,
		CCClient:      ccClient,
		PollInterval:  pollInterval,
		MetricsSender: metricsSender,
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

func (a *ASGSyncer) Poll() error {
	syncStartTime := time.Now()
	a.Logger.Debug("asg-sync-started")
	defer a.Logger.Debug("asg-sync-complete")

	token, err := a.UAAClient.GetToken()
	if err != nil {
		return err
	}

	retrieveStartTime := time.Now()
	ccSGs, err := a.CCClient.GetSecurityGroups(token)
	if err != nil {
		return err
	}
	retrieveEndTime := time.Now()
	a.MetricsSender.SendDuration(metricSecurityGroupsRetrievalFromCCDuration, retrieveEndTime.Sub(retrieveStartTime))

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

	syncEndTime := time.Now()
	a.MetricsSender.SendDuration(metricSecurityGroupsTotalSyncDuration, syncEndTime.Sub(syncStartTime))

	return err
}
