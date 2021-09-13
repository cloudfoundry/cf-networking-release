package cf_command

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

//go:generate counterfeiter -o ../fakes/security_group_cli_adapter.go --fake-name SecurityGroupCLIAdapter . securityGroupCLIAdapter
type securityGroupCLIAdapter interface {
	SecurityGroup(name string) (string, error)
}

type ASGChecker struct {
	Adapter securityGroupCLIAdapter
}

func (a *ASGChecker) CheckASG(name, expectedBody string) error {
	actualBody, err := a.Adapter.SecurityGroup(name)
	if err != nil {
		return fmt.Errorf("getting security group: %s", err)
	}

	match, err := compareASGs(actualBody, expectedBody)
	if err != nil {
		return err
	}

	if !match {
		return errors.New("security group mismatch")
	}

	return nil
}

func compareASGs(actualJSON, expectedJSON string) (bool, error) {
	var actual, expected interface{}

	if err := json.Unmarshal([]byte(actualJSON), &actual); err != nil {
		return false, fmt.Errorf("actual ASG is not valid JSON: %s", actualJSON)
	}
	if err := json.Unmarshal([]byte(expectedJSON), &expected); err != nil {
		return false, fmt.Errorf("expected ASG is not valid JSON: %s", expectedJSON)
	}

	return reflect.DeepEqual(actual, expected), nil
}
