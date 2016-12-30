package handlers

import (
	"fmt"
	"lib/models"
	"policy-server/uaa_client"
)

//go:generate counterfeiter -o ../fakes/policy_guard_cc_client.go --fake-name PolicyGuardCCClient . policyGuardCCClient
type policyGuardCCClient interface {
	GetSpaceGuids(token string, appGuids []string) ([]string, error)
	GetSpace(token, spaceGuid string) (models.Space, error)
	GetUserSpace(token, userGuid string, spaces models.Space) (models.Space, error)
}

type PolicyGuard struct {
	CCClient  policyGuardCCClient
	UAAClient uaaClient
}

func (g *PolicyGuard) CheckAccess(policies []models.Policy, userToken uaa_client.CheckTokenResponse) error {
	for _, scope := range userToken.Scope {
		if scope == "network.admin" {
			return nil
		}
	}
	token, err := g.UAAClient.GetToken()
	if err != nil {
		return fmt.Errorf("getting token: %s", err)
	}

	spaceGuids, err := g.CCClient.GetSpaceGuids(token, uniqueAppGuids(policies))
	if err != nil {
		return fmt.Errorf("getting space guids: %s", err)
	}
	var spaces []models.Space
	for _, guid := range spaceGuids {
		space, err := g.CCClient.GetSpace(token, guid)
		if err != nil {
			return fmt.Errorf("getting space with guid %s: %s", guid, err)
		}
		spaces = append(spaces, space)
	}
	var userSpaces []models.Space
	for _, space := range spaces {
		userSpace, err := g.CCClient.GetUserSpace(token, userToken.UserID, space)
		if err != nil {
			return fmt.Errorf("getting user space %s in org %s: %s", space.Name, space.OrgGuid, err)
		}
		userSpaces = append(userSpaces, userSpace)
	}
	return nil
}

func uniqueAppGuids(policies []models.Policy) []string {
	var set = make(map[string]struct{})
	for _, policy := range policies {
		set[policy.Source.ID] = struct{}{}
		set[policy.Destination.ID] = struct{}{}
	}
	var appGuids = make([]string, 0, len(set))
	for guid, _ := range set {
		appGuids = append(appGuids, guid)
	}
	return appGuids
}
