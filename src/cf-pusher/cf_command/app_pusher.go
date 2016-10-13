package cf_command

import (
	"fmt"
)

//go:generate counterfeiter -o ../fakes/push_cli_adapter.go --fake-name PushCLIAdapter . pushCLIAdapter
type pushCLIAdapter interface {
	Push(name, directory, manifestFile string) error
}

//go:generate counterfeiter -o ../fakes/manifest_generator.go --fake-name ManifestGenerator . manifestGenerator
type manifestGenerator interface {
	Generate(manifestStruct interface{}) (string, error)
}

type AppPusher struct {
	Applications      []Application
	Adapter           pushCLIAdapter
	ManifestGenerator manifestGenerator
}

type Application struct {
	Name      string
	Directory string
	Manifest  interface{}
}

func (a *AppPusher) Push() error {
	for _, app := range a.Applications {
		manifestFile := ""
		if app.Manifest == nil {
			manifestFile = fmt.Sprintf("%s/manifest.yml", app.Directory)
		} else {
			tmpManifest, err := a.ManifestGenerator.Generate(app.Manifest)
			if err != nil {
				return err
			}
			manifestFile = tmpManifest
		}
		err := a.Adapter.Push(app.Name, app.Directory, manifestFile)
		if err != nil {
			return err
		}
	}
	return nil
}
