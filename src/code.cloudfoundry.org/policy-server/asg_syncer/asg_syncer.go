package asg_syncer

import (
	"time"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/policy-server/cc_client"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/uaa_client"
)

type ASGSyncer struct {
	Logger         lager.Logger
	Store          store.SecurityGroupsStore
	UAAClient      uaa_client.UAAClient
	CCClient       cc_client.CCClient
	RequestTimeout time.Duration
}

func NewASGSyncer(logger lager.Logger, store store.SecurityGroupsStore, uaaClient uaa_client.UAAClient, ccClient cc_client.CCClient, requestTimeout time.Duration) *ASGSyncer {
	return &ASGSyncer{
		Logger:         logger,
		Store:          store,
		UAAClient:      uaaClient,
		CCClient:       ccClient,
		RequestTimeout: requestTimeout,
	}
}
