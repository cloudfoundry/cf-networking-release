package handlers

import "code.cloudfoundry.org/lager"

//go:generate counterfeiter -o ../fakes/client.go --fake-name Client . client
type client interface {
	GetSpaceGuids(appGuids []string) ([]string, error)
	GetSpaces(spaceGuids []string) ([]Space, error)
	GetUserSpaces(userGuid string, spaces []Space) ([]Space, error)
}

type Space struct {
	Name    string `json:name`
	OrgGuid string `json:organization_guid`
}

type SpaceGuard struct {
	Logger lager.Logger
	Client client
}
