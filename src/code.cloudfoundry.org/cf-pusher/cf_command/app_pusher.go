package cf_command

import (
	"encoding/json"
	"fmt"
	"time"
)

//go:generate counterfeiter -o ../fakes/push_cli_adapter.go --fake-name PushCLIAdapter . pushCLIAdapter
type pushCLIAdapter interface {
	CheckApp(guid string) ([]byte, error)
	AppGuid(name string) (string, error)
	Push(name, directory, manifestFile string) error
}

//go:generate counterfeiter -o ../fakes/manifest_generator.go --fake-name ManifestGenerator . manifestGenerator
type manifestGenerator interface {
	Generate(manifestStruct interface{}) (string, error)
}

type AppPusher struct {
	Applications            []Application
	Adapter                 pushCLIAdapter
	Concurrency             int
	ManifestPath            string
	Directory               string
	SkipIfPresent           bool
	DesiredRunningInstances int

	PushAttempts  int
	RetryWaitTime time.Duration
}

type Application struct {
	Name string
}

func (a *AppPusher) shouldPushApp(name string) bool {
	if a.SkipIfPresent {
		guid, err := a.Adapter.AppGuid(name)
		if err != nil {
			// App is not pushed yet
			return true
		}

		appBytes, err := a.Adapter.CheckApp(guid)
		if err != nil {
			// Error getting app summary
			return true
		}

		s := &AppStatus{}
		err = json.Unmarshal(appBytes, s)
		if err != nil || s.RunningInstances < a.DesiredRunningInstances {
			// Error unmarshalling response
			return true
		}

		return false

	}
	return true
}

func (a *AppPusher) Push() error {
	sem := make(chan bool, a.Concurrency)
	errs := make(chan error, len(a.Applications))

	for _, app := range a.Applications {
		sem <- true
		go func(o Application, m string) {
			defer func() { <-sem }()

			if a.shouldPushApp(o.Name) {
				var err error
				for attempt := 1; attempt <= 3; attempt++ {
					err = a.Adapter.Push(o.Name, a.Directory, m)
					if err == nil {
						break
					}

					if attempt < a.PushAttempts {
						fmt.Printf("Failed to push app '%s' on attempt number %d. Retrying...\n", o.Name, attempt)
						time.Sleep(a.RetryWaitTime)
					} else {
						fmt.Printf("Failed to push app '%s' on attempt number %d. Max attempts reached. Bailing...\n", o.Name, attempt)
					}
				}

				if err != nil {
					errs <- err
				}
			}

		}(app, a.ManifestPath)
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
