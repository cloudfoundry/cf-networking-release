package cf_command

//go:generate counterfeiter -o ../fakes/push_cli_adapter.go --fake-name PushCLIAdapter . pushCLIAdapter
type pushCLIAdapter interface {
	AppGuid(name string) (string, error)
	Push(name, directory, manifestFile string) error
}

//go:generate counterfeiter -o ../fakes/manifest_generator.go --fake-name ManifestGenerator . manifestGenerator
type manifestGenerator interface {
	Generate(manifestStruct interface{}) (string, error)
}

type AppPusher struct {
	Applications  []Application
	Adapter       pushCLIAdapter
	Concurrency   int
	ManifestPath  string
	Directory     string
	SkipIfPresent bool
}

type Application struct {
	Name string
}

func (a *AppPusher) shouldPushApp(name string) bool {
	if a.SkipIfPresent {
		_, err := a.Adapter.AppGuid(name)
		if err == nil {
			return false
		}
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
				err := a.Adapter.Push(o.Name, a.Directory, m)
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
