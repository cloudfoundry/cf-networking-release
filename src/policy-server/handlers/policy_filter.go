package handlers

import (
	"fmt"
	"policy-server/api"
	"policy-server/uaa_client"
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
	GetUserSpace(token, userGUID string, spaces api.Space) (*api.Space, error)
	GetUserSpaces(token, userGUID string) (map[string]struct{}, error)
}

type PolicyFilter struct {
	CCClient  ccClient
	UAAClient uaaClient
	ChunkSize int
}

func (f *PolicyFilter) FilterPolicies(policies []api.Policy, userToken uaa_client.CheckTokenResponse) ([]api.Policy, error) {
	for _, scope := range userToken.Scope {
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

	userSpaces, err := f.CCClient.GetUserSpaces(token, userToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("getting user spaces: %s", err)
	}

	filtered := filter(policies, appSpaces, userSpaces)

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

func filter(policies []api.Policy, appSpaces map[string]string, userSpaces map[string]struct{}) []api.Policy {
	filtered := []api.Policy{}

	for _, policy := range policies {
		_, sourceFound := userSpaces[appSpaces[policy.Source.ID]]
		_, destFound := userSpaces[appSpaces[policy.Destination.ID]]
		if sourceFound && destFound {
			filtered = append(filtered, policy)
		}
	}
	return filtered
}
