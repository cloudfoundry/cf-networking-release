package handlers

import (
	"fmt"

	"code.cloudfoundry.org/policy-server/cc_client"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/uaa_client"
)

type PolicyGuard struct {
	CCClient  cc_client.CCClient
	UAAClient uaa_client.UAAClient
}

func NewPolicyGuard(uaaClient uaa_client.UAAClient, ccClient cc_client.CCClient) *PolicyGuard {
	return &PolicyGuard{
		CCClient:  ccClient,
		UAAClient: uaaClient,
	}
}

func (g *PolicyGuard) CheckAccess(policies []store.Policy, subjectToken uaa_client.CheckTokenResponse) (bool, error) {
	for _, scope := range subjectToken.Scope {
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
		subjectSpace, err := g.CCClient.GetSubjectSpace(token, subjectToken.Subject, *space)
		if err != nil {
			return false, fmt.Errorf("getting space with guid %s: %s", guid, err)
		}
		if subjectSpace == nil {
			return false, nil
		}
	}
	return true, nil
}

func (g *PolicyGuard) IsNetworkAdmin(subjectToken uaa_client.CheckTokenResponse) bool {
	for _, scope := range subjectToken.Scope {
		if scope == "network.admin" {
			return true
		}
	}

	return false
}

func uniqueAppGUIDs(policies []store.Policy) []string {
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
