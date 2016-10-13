package cf_cli_adapter

import (
	"fmt"
	"os/exec"
)

type Adapter struct {
	CfCliPath string
}

func (a *Adapter) CreateOrg(name string) error {
	return exec.Command(a.CfCliPath, "create-org", name).Run()
}

func (a *Adapter) CreateSpace(name string) error {
	return exec.Command(a.CfCliPath, "create-space", name).Run()
}

func (a *Adapter) TargetOrg(name string) error {
	return exec.Command(a.CfCliPath, "target", "-o", name).Run()
}

func (a *Adapter) TargetSpace(name string) error {
	return exec.Command(a.CfCliPath, "target", "-s", name).Run()
}

func (a *Adapter) SetApiWithSsl(api string) error {
	return exec.Command(a.CfCliPath, "api", api).Run()
}

func (a *Adapter) SetApiWithoutSsl(api string) error {
	return exec.Command(a.CfCliPath, "api", api, "--skip-ssl-validation").Run()
}

func (a *Adapter) Auth(user, password string) error {
	return exec.Command(a.CfCliPath, "auth", user, password).Run()
}

func (a *Adapter) Push(name, directory, manifestFile string) error {
	fmt.Printf("running: %s push %s -p %s -f %s\n", a.CfCliPath, name, directory, manifestFile)
	bytes, err := exec.Command(a.CfCliPath,
		"push", name,
		"-p", directory,
		"-f", manifestFile).CombinedOutput()
	fmt.Printf("output: %s\n", string(bytes))
	return err
}
