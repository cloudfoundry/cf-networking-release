package cf_command

import "fmt"

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
	Concurrency       int
}

type Application struct {
	Name      string
	Directory string
	Manifest  interface{}
}

func (a *AppPusher) Push() error {
	sem := make(chan bool, a.Concurrency)
	errs := make(chan error, len(a.Applications))
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
		sem <- true
		go func(o Application, m string) {
			defer func() { <-sem }()
			err := a.Adapter.Push(o.Name, o.Directory, m)
			if err != nil {
				errs <- err
			}
		}(app, manifestFile)
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
