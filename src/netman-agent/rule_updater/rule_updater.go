package rule_updater

import (
	"fmt"
	"netman-agent/models"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/store_reader.go --fake-name StoreReader . storeReader
type storeReader interface {
	GetContainers() models.Containers
}

//go:generate counterfeiter -o ../fakes/policy_client.go --fake-name PolicyClient . policyClient
type policyClient interface {
	GetPolicies() ([]models.Policy, error)
}

type Updater struct {
	Logger       lager.Logger
	storeReader  storeReader
	policyClient policyClient
}

func New(logger lager.Logger, storeReader storeReader, policyClient policyClient) *Updater {
	return &Updater{
		Logger:       logger,
		storeReader:  storeReader,
		policyClient: policyClient,
	}
}

func (u *Updater) Update() error {
	containers := u.storeReader.GetContainers()
	policies, err := u.policyClient.GetPolicies()
	if err != nil {
		u.Logger.Error("get-policies", err)
		return fmt.Errorf("get policies failed: %s", err)
	}

	//local
	for _, policy := range policies {
		srcContainers, srcOk := containers[policy.Source.ID]
		dstContainers, dstOk := containers[policy.Destination.ID]

		if srcOk && dstOk {
			for _, srcContainer := range srcContainers {
				for _, dstContainer := range dstContainers {
					u.Logger.Info("enforce-local-rule", lager.Data{
						"srcIP": srcContainer.IP,
						"dstIP": dstContainer.IP,
						"port":  policy.Destination.Port,
						"proto": policy.Destination.Protocol,
					})
				}
			}
		}
	}

	return nil
}
