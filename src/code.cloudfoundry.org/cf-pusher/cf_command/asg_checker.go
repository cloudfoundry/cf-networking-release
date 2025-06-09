package cf_command

import (
	"encoding/json"
	"errors"
	"fmt"

	"code.cloudfoundry.org/cf-pusher/cf_cli_adapter"
)

//go:generate counterfeiter -o ../fakes/security_group_cli_adapter.go --fake-name SecurityGroupCLIAdapter . securityGroupCLIAdapter
type securityGroupCLIAdapter interface {
	SecurityGroup(name string) (cf_cli_adapter.ASG, error)
}

type ASGChecker struct {
	Adapter securityGroupCLIAdapter
}

func (a *ASGChecker) CheckASG(name, expectedRules string, expectedGloballyRunning, expectedGloballyStaging bool) error {
	asg, err := a.Adapter.SecurityGroup(name)
	if err != nil {
		return fmt.Errorf("getting security group: %s", err)
	}

	match, err := compareASGs(asg, expectedRules)
	if err != nil {
		return err
	}

	if !match {
		return errors.New("security group rules mismatch")
	}
	if asg.GloballyEnabled.Running != expectedGloballyRunning {
		return errors.New("security group globally-enabled-running mismatch")
	}

	if asg.GloballyEnabled.Staging != expectedGloballyStaging {
		return errors.New("security group globally-enabled-staging mismatch")
	}
	return nil
}

func compareASGs(actualASG cf_cli_adapter.ASG, expectedJSON string) (bool, error) {
	var expected []cf_cli_adapter.ASGRule

	if err := json.Unmarshal([]byte(expectedJSON), &expected); err != nil {
		return false, fmt.Errorf("expected ASG is not valid JSON: %s", expectedJSON)
	}

	return len(actualASG.Rules) == len(expected), nil
}
