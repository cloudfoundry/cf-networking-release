package handlers

import (
	"fmt"

	"code.cloudfoundry.org/policy-server/cc_client"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/uaa_client"
)

type PolicyFilter struct {
	CCClient  cc_client.CCClient
	UAAClient uaa_client.UAAClient
	ChunkSize int
}

func NewPolicyFilter(uaaClient uaa_client.UAAClient, ccClient cc_client.CCClient, chunkSize int) *PolicyFilter {
	return &PolicyFilter{
		CCClient:  ccClient,
		UAAClient: uaaClient,
		ChunkSize: chunkSize,
	}
}

func (f *PolicyFilter) FilterPolicies(policies []store.Policy, subjectToken uaa_client.CheckTokenResponse) ([]store.Policy, error) {
	for _, scope := range subjectToken.Scope {
		if scope == "network.admin" {
			return policies, nil
		}
	}

	token, err := f.UAAClient.GetToken()
	if err != nil {
		return nil, fmt.Errorf("getting token: %s", err)
	}

	appGuids := uniqueAppGUIDs(policies)
	appGuidChunks := getChunks(appGuids, f.ChunkSize)

	appSpacesList := []map[string]string{}
	for _, chunk := range appGuidChunks {
		spaces, err := f.CCClient.GetAppSpaces(token, chunk)
		if err != nil {
			return nil, fmt.Errorf("getting app spaces: %s", err)
		}
		appSpacesList = append(appSpacesList, spaces)
	}

	appSpaces := flatten(appSpacesList)

	subjectSpaces, err := f.CCClient.GetSubjectSpaces(token, subjectToken.Subject)
	if err != nil {
		return nil, fmt.Errorf("getting subject spaces: %s", err)
	}

	filtered := filter(policies, appSpaces, subjectSpaces)

	return filtered, nil
}

func flatten(list []map[string]string) map[string]string {
	ret := make(map[string]string)
	for _, m := range list {
		for k, v := range m {
			ret[k] = v
		}
	}
	return ret
}

func getChunks(appGuids []string, chunkSize int) [][]string {
	if chunkSize < 1 {
		chunkSize = 100
	}
	appGuidChunks := [][]string{}
	for i := 0; i < len(appGuids); i += chunkSize {
		last := i + chunkSize
		if last > len(appGuids) {
			last = len(appGuids)
		}
		appGuidChunks = append(appGuidChunks, appGuids[i:last])
	}

	return appGuidChunks
}

func filter(policies []store.Policy, appSpaces map[string]string, subjectSpaces map[string]struct{}) []store.Policy {
	filtered := []store.Policy{}

	for _, policy := range policies {
		_, sourceFound := subjectSpaces[appSpaces[policy.Source.ID]]
		_, destFound := subjectSpaces[appSpaces[policy.Destination.ID]]
		if sourceFound && destFound {
			filtered = append(filtered, policy)
		}
	}
	return filtered
}
