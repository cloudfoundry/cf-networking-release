package cf_command

import (
	"encoding/json"
	"errors"
	"fmt"
)

//go:generate counterfeiter -o ../fakes/check_cli_adapter.go --fake-name CheckCLIAdapter . checkCLIAdapter
type checkCLIAdapter interface {
	OrgGuid(name string) (string, error)
	AppCount(orgGuid string) (int, error)
	CheckApp(guid string) ([]byte, error)
	AppGuid(name string) (string, error)
}

type AppChecker struct {
	Org          string
	Applications []Application
	Adapter      checkCLIAdapter
}

type AppStatus struct {
	GUID             string `json:"guid"`
	Name             string `json:"name"`
	RunningInstances int    `json:"running_instances"`
	Instances        int    `json:"instances"`
	State            string `json:"state"`
}

func (a *AppChecker) CheckApps() error {
	orgGuid, err := a.Adapter.OrgGuid(a.Org)
	if err != nil {
		return fmt.Errorf("checking org guid %s: %s", a.Org, err)
	}
	appCount, err := a.Adapter.AppCount(orgGuid)
	if err != nil {
		return fmt.Errorf("checking app counts: %s", err)
	}
	if appCount != len(a.Applications) {
		return errors.New(fmt.Sprintf("app count %d does not match %d", appCount, len(a.Applications)))
	}

	for _, app := range a.Applications {
		guid, err := a.Adapter.AppGuid(app.Name)
		if err != nil {
			return fmt.Errorf("checking app guid %s: %s", app.Name, err)
		}
		result, err := a.Adapter.CheckApp(guid)
		if err != nil {
			return fmt.Errorf("checking app %s: %s", app.Name, err)
		}

		s := &AppStatus{}
		if err := json.Unmarshal(result, s); err != nil {
			return err
		}

		if s.Instances == 0 {
			return fmt.Errorf("checking app %s: %s", app.Name, "no instances are running")
		}

		if s.RunningInstances != s.Instances {
			return fmt.Errorf("checking app %s: %s", app.Name, "not all instances are running")
		}
	}
	return nil
}
