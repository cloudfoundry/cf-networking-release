package cf_command

//go:generate counterfeiter -o ../fakes/org_checker_cli.go --fake-name OrgCheckerCli . orgCheckerCli
type orgCheckerCli interface {
	TargetOrg(name string) error
}

type OrgChecker struct {
	Org     string
	Adapter orgCheckerCli
}

func (c *OrgChecker) CheckOrgExists() bool {
	err := c.Adapter.TargetOrg(c.Org)
	if err != nil {
		return false
	}

	return true
}
