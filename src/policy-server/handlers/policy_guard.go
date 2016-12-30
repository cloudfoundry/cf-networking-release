package handlers

import (
	"fmt"
	"lib/models"
	"policy-server/uaa_client"
)

//go:generate counterfeiter -o ../fakes/policy_guard_cc_client.go --fake-name PolicyGuardCCClient . policyGuardCCClient
type policyGuardCCClient interface {
	GetSpaceGuids(token string, appGuids []string) ([]string, error)
	GetSpaces(token string, spaceGuids []string) ([]models.Space, error)
	GetUserSpaces(token, userGuid string, spaces []models.Space) ([]models.Space, error)
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

	spaceGuids, err := g.CCClient.GetSpaceGuids(token, uniqueAppGuids(policies))
	if err != nil {
		return false, fmt.Errorf("getting space guids: %s", err)
	}
	spaces, err := g.CCClient.GetSpaces(token, spaceGuids)
	if err != nil {
		return false, fmt.Errorf("getting spaces: %s", err)
	}
	userSpaces, err := g.CCClient.GetUserSpaces(token, userToken.UserID, spaces)
	if err != nil {
		return false, fmt.Errorf("getting user spaces: %s", err)
	}
	if len(spaces) != len(userSpaces) {
		return false, nil
	}
	return true, nil
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
