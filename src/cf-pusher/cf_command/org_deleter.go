package cf_command

//go:generate counterfeiter -o ../fakes/org_deleter_cli.go --fake-name OrgDeleterCli . orgDeleterCli
type orgDeleterCli interface {
	DeleteOrg(name string) error
}

type OrgDeleter struct {
	Org     string
	Adapter orgDeleterCli
}

func (c *OrgDeleter) Delete() error {
	return c.Adapter.DeleteOrg(c.Org)
}
