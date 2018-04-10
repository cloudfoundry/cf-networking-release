package lib

import "os/exec"

//go:generate counterfeiter -o fakes/command_runner.go --fake-name CommandRunner . CommandRunner
type CommandRunner interface {
	Exec(name string, arg ...string) ([]byte, error)
}

type RealCommandRunner struct {
}

func (RealCommandRunner) Exec(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}
