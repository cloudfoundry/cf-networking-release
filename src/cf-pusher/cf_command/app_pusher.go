package cf_command

//go:generate counterfeiter -o ../fakes/push_cli_adapter.go --fake-name PushCLIAdapter . pushCLIAdapter
type pushCLIAdapter interface {
	Push(name, directory, manifestFile string) error
}

//go:generate counterfeiter -o ../fakes/manifest_generator.go --fake-name ManifestGenerator . manifestGenerator
type manifestGenerator interface {
	Generate(manifestStruct interface{}) (string, error)
}

type AppPusher struct {
	Applications []Application
	Adapter      pushCLIAdapter
	Concurrency  int
	ManifestPath string
	Directory    string
}

type Application struct {
	Name string
}

func (a *AppPusher) Push() error {
	sem := make(chan bool, a.Concurrency)
	errs := make(chan error, len(a.Applications))

	for _, app := range a.Applications {
		sem <- true
		go func(o Application, m string) {
			defer func() { <-sem }()
			err := a.Adapter.Push(o.Name, a.Directory, m)
			if err != nil {
				errs <- err
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
