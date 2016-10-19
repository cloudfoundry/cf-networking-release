package cf_cli_adapter

import (
	"fmt"
	"os/exec"
	"strings"
)

type Adapter struct {
	CfCliPath string
}

func (a *Adapter) CreateOrg(name string) error {
	fmt.Printf("running: %s create-org %s\n", a.CfCliPath, name)
	return exec.Command(a.CfCliPath, "create-org", name).Run()
}

func (a *Adapter) DeleteOrg(name string) error {
	fmt.Printf("running: %s delete-org -f %s\n", a.CfCliPath, name)
	return exec.Command(a.CfCliPath, "delete-org", "-f", name).Run()
}

func (a *Adapter) CreateSpace(name string) error {
	fmt.Printf("running: %s create-space %s\n", a.CfCliPath, name)
	return exec.Command(a.CfCliPath, "create-space", name).Run()
}

func (a *Adapter) TargetOrg(name string) error {
	fmt.Printf("running: %s target -o  %s\n", a.CfCliPath, name)
	return exec.Command(a.CfCliPath, "target", "-o", name).Run()
}

func (a *Adapter) TargetSpace(name string) error {
	fmt.Printf("running: %s target -s  %s\n", a.CfCliPath, name)
	return exec.Command(a.CfCliPath, "target", "-s", name).Run()
}

func (a *Adapter) SetApiWithSsl(api string) error {
	fmt.Printf("running: %s api  %s\n", a.CfCliPath, api)
	return exec.Command(a.CfCliPath, "api", api).Run()
}

func (a *Adapter) SetApiWithoutSsl(api string) error {
	fmt.Printf("running: %s api  %s --skip-ssl-validation\n", a.CfCliPath, api)
	return exec.Command(a.CfCliPath, "api", api, "--skip-ssl-validation").Run()
}

func (a *Adapter) Auth(user, password string) error {
	fmt.Printf("running: %s auth <user> <pass> \n", a.CfCliPath)
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

func (a *Adapter) AppGuid(name string) (string, error) {
	fmt.Printf("running: %s app %s --guid\n", a.CfCliPath, name)
	bytes, err := exec.Command(a.CfCliPath, "app", name, "--guid").CombinedOutput()
	return strings.TrimSpace(string(bytes)), err
}

func (a *Adapter) CheckApp(guid string) ([]byte, error) {
	fmt.Printf("running: %s curl \"/v2/apps/%s/summary/\"\n", a.CfCliPath, guid)
	bytes, err := exec.Command(a.CfCliPath, "curl", fmt.Sprintf("/v2/apps/%s/summary", guid)).CombinedOutput()
	return bytes, err
}

func (a *Adapter) AccessAllow(sourceApp, destApp string, port int, protocol string) error {
	portStr := fmt.Sprintf("%d", port)
	fmt.Printf("running: cf access-allow %s %s --port %s --protocol tcp\n", sourceApp, destApp, portStr)
	return exec.Command("cf", "access-allow", sourceApp, destApp, "--port", portStr, "--protocol", "tcp").Run()
}

func (a *Adapter) AccessDeny(sourceApp, destApp string, port int, protocol string) error {
	portStr := fmt.Sprintf("%d", port)
	fmt.Printf("running: cf access-deny %s %s --port %s --protocol tcp\n", sourceApp, destApp, portStr)
	return exec.Command("cf", "access-deny", sourceApp, destApp, "--port", portStr, "--protocol", "tcp").Run()
}
