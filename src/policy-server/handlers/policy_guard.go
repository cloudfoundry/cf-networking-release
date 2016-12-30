package handlers

import (
	"fmt"
	"policy-server/models"
	"policy-server/uaa_client"
)

//go:generate counterfeiter -o ../fakes/policy_guard_cc_client.go --fake-name PolicyGuardCCClient . policyGuardCCClient
type policyGuardCCClient interface {
	GetSpaceGUIDs(token string, appGUIDs []string) ([]string, error)
	GetSpace(token, spaceGUID string) (*models.Space, error)
	GetUserSpace(token, userGUID string, spaces models.Space) (*models.Space, error)
}

type PolicyGuard struct {
	CCClient  policyGuardCCClient
	UAAClient uaaClient
}

func (g *PolicyGuard) CheckAccess(policies []models.Policy, userToken uaa_client.CheckTokenResponse) (bool, error) {
	for _, scope := range userToken.Scope {
		if scope == "network.admin" {
			return true, nil
		}
	}
	token, err := g.UAAClient.GetToken()
	if err != nil {
		return false, fmt.Errorf("getting token: %s", err)
	}

	spaceGUIDs, err := g.CCClient.GetSpaceGUIDs(token, uniqueAppGUIDs(policies))
	if err != nil {
		return false, fmt.Errorf("getting space guids: %s", err)
	}
	for _, guid := range spaceGUIDs {
		space, err := g.CCClient.GetSpace(token, guid)
		if err != nil {
			return false, fmt.Errorf("getting space with guid %s: %s", guid, err)
		}
		if space == nil {
			return false, nil
		}
		userSpace, err := g.CCClient.GetUserSpace(token, userToken.UserID, *space)
		if err != nil {
			return false, fmt.Errorf("getting space with guid %s: %s", guid, err)
		}
		if userSpace == nil {
			return false, nil
		}
	}
	return true, nil
}

func uniqueAppGUIDs(policies []models.Policy) []string {
	var set = make(map[string]struct{})
	for _, policy := range policies {
		set[policy.Source.ID] = struct{}{}
		set[policy.Destination.ID] = struct{}{}
	}
	var appGUIDs = make([]string, 0, len(set))
	for guid, _ := range set {
		appGUIDs = append(appGUIDs, guid)
	}
	return appGUIDs
}
