package handlers

import (
	"fmt"
	"policy-server/store"
	"policy-server/uaa_client"
)

type QuotaGuard struct {
	Store       store.Store
	MaxPolicies int
}

func NewQuotaGuard(store store.Store, maxPolicies int) *QuotaGuard {
	return &QuotaGuard{
		Store:       store,
		MaxPolicies: maxPolicies,
	}
}

func (g *QuotaGuard) CheckAccess(policyCollection store.PolicyCollection, userToken uaa_client.CheckTokenResponse) (bool, error) {
	for _, scope := range userToken.Scope {
		if scope == "network.admin" {
			return true, nil
		}
	}

	if len(policyCollection.EgressPolicies) > 0 {
		return false, nil
	}

	appGuids := uniqueAppGUIDs(policyCollection.Policies)
	toAddSourceCounts := sourceCounts(policyCollection.Policies, appGuids)
	sourcePolicies, err := g.Store.ByGuids(appGuids, []string{}, false)
	if err != nil {
		return false, fmt.Errorf("getting policies: %s", err)
	}
	currentAppCounts := sourceCounts(sourcePolicies, appGuids)
	for _, appGuid := range appGuids {
		if currentAppCounts[appGuid]+toAddSourceCounts[appGuid] > g.MaxPolicies {
			return false, nil
		}
	}
	return true, nil
}

func sourceCounts(policies []store.Policy, knownAppGuids []string) map[string]int {
	var set = make(map[string]int)
	for _, appGuid := range knownAppGuids {
		set[appGuid] = 0
	}
	for _, policy := range policies {
		set[policy.Source.ID] = set[policy.Source.ID] + 1
	}
	return set
}
