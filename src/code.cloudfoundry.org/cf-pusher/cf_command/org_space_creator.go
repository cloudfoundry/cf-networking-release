package cf_command

import "fmt"

//go:generate counterfeiter -o ../fakes/org_space_cli.go --fake-name OrgSpaceCli . orgSpaceCli
type orgSpaceCli interface {
	CreateOrg(name string) error
	CreateSpace(spaceName, orgName string) error
	TargetOrg(name string) error
	TargetSpace(name string) error
	CreateQuota(name, memory string, instanceMemory, routes, serviceInstances, appInstances, routePorts int) error
	SetQuota(org, quota string) error
}

type Quota struct {
	Name             string
	Memory           string
	InstanceMemory   int
	Routes           int
	ServiceInstances int
	AppInstances     int
	RoutePorts       int
}

type OrgSpaceCreator struct {
	Org     string
	Space   string
	Quota   Quota
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

	err = c.Adapter.CreateSpace(c.Space, c.Org)
	if err != nil {
		return fmt.Errorf("creating space: %s", err)
	}

	err = c.Adapter.TargetSpace(c.Space)
	if err != nil {
		return fmt.Errorf("targeting space: %s", err)
	}

	q := c.Quota
	err = c.Adapter.CreateQuota(q.Name, q.Memory, q.InstanceMemory, q.Routes, q.ServiceInstances, q.AppInstances, q.RoutePorts)
	if err != nil {
		return fmt.Errorf("creating quota: %s", err)
	}

	err = c.Adapter.SetQuota(c.Org, c.Quota.Name)
	if err != nil {
		return fmt.Errorf("setting quota: %s", err)
	}

	return nil
}
