package cf_command

import "fmt"

//go:generate counterfeiter -o ../fakes/security_group_installation_cli_adapter.go --fake-name SecurityGroupInstallationCLIAdapter . securityGroupInstallationCLIAdapter
type securityGroupInstallationCLIAdapter interface {
	DeleteSecurityGroup(name string) error
	CreateSecurityGroup(name, body string) error
	BindSecurityGroup(asgName, orgName, spaceName string) error
}

type ASGInstaller struct {
	Adapter securityGroupInstallationCLIAdapter
}

func (a *ASGInstaller) InstallASG(asgName, asgFilePath, orgName, spaceName string) error {
	if err := a.Adapter.DeleteSecurityGroup(asgName); err != nil {
		return fmt.Errorf("deleting security group: %s", err)
	}
	if err := a.Adapter.CreateSecurityGroup(asgName, asgFilePath); err != nil {
		return fmt.Errorf("creating security group: %s", err)
	}
	if err := a.Adapter.BindSecurityGroup(asgName, orgName, spaceName); err != nil {
		return fmt.Errorf("binding security group: %s", err)
	}
	return nil
}
