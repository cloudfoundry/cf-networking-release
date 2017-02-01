package handlers

import (
	"fmt"
	"policy-server/models"
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
	GetSpace(token, spaceGUID string) (*models.Space, error)
	GetSpaceGUIDs(token string, appGUIDs []string) ([]string, error)
	GetUserSpace(token, userGUID string, spaces models.Space) (*models.Space, error)
	GetUserSpaces(token, userGUID string) (map[string]struct{}, error)
}

type PolicyFilter struct {
	CCClient  ccClient
	UAAClient uaaClient
}

func (g *PolicyFilter) FilterPolicies(policies []models.Policy, userToken uaa_client.CheckTokenResponse) ([]models.Policy, error) {
	for _, scope := range userToken.Scope {
		if scope == "network.admin" {
			return policies, nil
		}
	}

	token, err := g.UAAClient.GetToken()
	if err != nil {
		return nil, fmt.Errorf("getting token: %s", err)
	}

	appGuids := uniqueAppGUIDs(policies)
	appSpaces, err := g.CCClient.GetAppSpaces(token, appGuids)
	if err != nil {
		return nil, fmt.Errorf("getting app spaces: %s", err)
	}

	userSpaces, err := g.CCClient.GetUserSpaces(token, userToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("getting user spaces: %s", err)
	}

	filtered := filter(policies, appSpaces, userSpaces)

	return filtered, nil
}

func filter(policies []models.Policy, appSpaces map[string]string, userSpaces map[string]struct{}) []models.Policy {
	filtered := []models.Policy{}

	for _, policy := range policies {
		_, sourceFound := userSpaces[appSpaces[policy.Source.ID]]
		_, destFound := userSpaces[appSpaces[policy.Destination.ID]]
		if sourceFound && destFound {
			filtered = append(filtered, policy)
		}
	}
	return filtered
}
