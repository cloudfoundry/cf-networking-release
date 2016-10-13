package cf_command

import "fmt"

//go:generate counterfeiter -o ../fakes/org_space_cli.go --fake-name OrgSpaceCli . orgSpaceCli
type orgSpaceCli interface {
	CreateOrg(name string) error
	CreateSpace(name string) error
	TargetOrg(name string) error
	TargetSpace(name string) error
}

type OrgSpaceCreator struct {
	Org     string
	Space   string
	Adapter orgSpaceCli
}

func (c *OrgSpaceCreator) Create() error {
	err := c.Adapter.CreateOrg(c.Org)
	if err != nil {
		return fmt.Errorf("creating org: %s", err)
	}

	err = c.Adapter.TargetOrg(c.Org)
	if err != nil {
		return fmt.Errorf("targeting org: %s", err)
	}

	err = c.Adapter.CreateSpace(c.Space)
	if err != nil {
		return fmt.Errorf("creating space: %s", err)
	}

	err = c.Adapter.TargetSpace(c.Space)
	if err != nil {
		return fmt.Errorf("targeting space: %s", err)
	}
	return nil
}
