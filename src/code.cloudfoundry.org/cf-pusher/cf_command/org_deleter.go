package cf_command

import "fmt"

//go:generate counterfeiter -o ../fakes/org_deleter_cli.go --fake-name OrgDeleterCli . orgDeleterCli
type orgDeleterCli interface {
	DeleteOrg(name string) error
	DeleteQuota(name string) error
}

type OrgDeleter struct {
	Org     string
	Quota   Quota
	Adapter orgDeleterCli
}

func (c *OrgDeleter) Delete() error {
	err := c.Adapter.DeleteOrg(c.Org)
	if err != nil {
		return fmt.Errorf("deleting org: %s", err)
	}
	err = c.Adapter.DeleteQuota(c.Quota.Name)
	if err != nil {
		return fmt.Errorf("deleting quota: %s", err)
	}
	return nil
}
