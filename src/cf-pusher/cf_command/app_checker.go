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
	Concurrency  int
}

type AppStatus struct {
	GUID             string `json:"guid"`
	Name             string `json:"name"`
	RunningInstances int    `json:"running_instances"`
	Instances        int    `json:"instances"`
	State            string `json:"state"`
}

func (a *AppChecker) CheckApps(appSpec map[string]int) error {
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

	sem := make(chan bool, a.Concurrency)
	errs := make(chan error, len(a.Applications))
	for _, o := range a.Applications {
		sem <- true
		go func(app Application) {
			defer func() { <-sem }()
			guid, err := a.Adapter.AppGuid(app.Name)
			if err != nil {
				errs <- fmt.Errorf("checking app guid %s: %s", app.Name, err)
				return
			}
			result, err := a.Adapter.CheckApp(guid)
			if err != nil {
				errs <- fmt.Errorf("checking app %s: %s", app.Name, err)
				return
			}

			s := &AppStatus{}
			if err := json.Unmarshal(result, s); err != nil {
				errs <- err
				return
			}

			if s.Instances == 0 {
				errs <- fmt.Errorf("checking app %s: %s", app.Name, "no instances are running")
				return
			}

			if s.RunningInstances != s.Instances {
				errs <- fmt.Errorf("checking app %s: %s", app.Name, "not all instances are running")
				return
			}

			if desiredInstances, ok := appSpec[app.Name]; ok {
				if appSpec[app.Name] != s.RunningInstances {
					errs <- fmt.Errorf("checking app %s: %s, running: %d desired: %d", app.Name, "not running desired instances", s.RunningInstances, desiredInstances)
					return
				}
			} else {
				errs <- fmt.Errorf("checking app %s: not found in app spec", app.Name)
				return
			}
		}(o)
	}

	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

	close(errs)
	if err := <-errs; err != nil {
		return err
	}

	return nil
}
