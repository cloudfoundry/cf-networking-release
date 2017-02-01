package handlers

import (
	"fmt"
	"policy-server/models"
	"policy-server/store"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/uua_client.go --fake-name UAAClient . uaaClient
type uaaClient interface {
	GetToken() (string, error)
}

//go:generate counterfeiter -o ../fakes/cc_client.go --fake-name CCClient . ccClient
type ccClient interface {
	GetAllAppGUIDs(token string) (map[string]interface{}, error)
}

type PolicyCleaner struct {
	Logger    lager.Logger
	Store     store.Store
	UAAClient uaaClient
	CCClient  ccClient
}

func (p *PolicyCleaner) DeleteStalePolicies() ([]models.Policy, error) {
	policies, err := p.Store.All()
	if err != nil {
		p.Logger.Error("store-list-policies-failed", err)
		return nil, fmt.Errorf("database read failed: %s", err)
	}

	token, err := p.UAAClient.GetToken()
	if err != nil {
		p.Logger.Error("get-uaa-token-failed", err)
		return nil, fmt.Errorf("get UAA token failed: %s", err)
	}

	ccAppGuids, err := p.CCClient.GetAllAppGUIDs(token)
	if err != nil {
		p.Logger.Error("cc-get-app-guids-failed", err)
		return nil, fmt.Errorf("get app guids from Cloud-Controller failed: %s", err)
	}

	stalePolicies := getStalePolicies(policies, ccAppGuids)

	p.Logger.Info("deleting stale policies:", lager.Data{
		"total_policies": len(stalePolicies),
		"stale_policies": stalePolicies,
	})
	err = p.Store.Delete(stalePolicies)
	if err != nil {
		p.Logger.Error("store-delete-policies-failed", err)
		return nil, fmt.Errorf("database write failed: %s", err)
	}

	return stalePolicies, nil
}

func getStalePolicies(policyList []models.Policy, ccList map[string]interface{}) (stalePolicies []models.Policy) {
	for _, p := range policyList {
		_, foundSrc := ccList[p.Source.ID]
		_, foundDst := ccList[p.Destination.ID]
		if !foundSrc || !foundDst {
			stalePolicies = append(stalePolicies, p)
		}
	}
	return stalePolicies
}
