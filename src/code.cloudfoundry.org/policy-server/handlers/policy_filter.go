package handlers

import (
	"fmt"

	"code.cloudfoundry.org/policy-server/api"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/uaa_client"
)

//go:generate counterfeiter -o fakes/uua_client.go --fake-name UAAClient . uaaClient
type uaaClient interface {
	GetToken() (string, error)
	CheckToken(string) (uaa_client.CheckTokenResponse, error)
}

//go:generate counterfeiter -o fakes/cc_client.go --fake-name CCClient . ccClient
type ccClient interface {
	GetAppSpaces(token string, appGUIDs []string) (map[string]string, error)
	GetSpace(token, spaceGUID string) (*api.Space, error)
	GetSpaceGUIDs(token string, appGUIDs []string) ([]string, error)
	GetSubjectSpace(token, subjectId string, spaces api.Space) (*api.Space, error)
	GetSubjectSpaces(token, subjectId string) (map[string]struct{}, error)
}

type PolicyFilter struct {
	CCClient  ccClient
	UAAClient uaaClient
	ChunkSize int
}

func NewPolicyFilter(uaaClient uaaClient, ccClient ccClient, chunkSize int) *PolicyFilter {
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
