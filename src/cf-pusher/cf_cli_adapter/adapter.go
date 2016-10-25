package cf_cli_adapter

import (
	"encoding/json"
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

func (a *Adapter) Scale(name string, instances int) error {
	instancesStr := fmt.Sprintf("%d", instances)
	fmt.Printf("running: %s scale %s -i %s\n", a.CfCliPath, name, instancesStr)
	err := exec.Command(a.CfCliPath, "scale", name, "-i", instancesStr).Run()
	return err
}

func (a *Adapter) AppGuid(name string) (string, error) {
	fmt.Printf("running: %s app %s --guid\n", a.CfCliPath, name)
	bytes, err := exec.Command(a.CfCliPath, "app", name, "--guid").CombinedOutput()
	return strings.TrimSpace(string(bytes)), err
}

type Apps struct {
	TotalResults int `json:"total_results"`
}

func (a *Adapter) OrgGuid(name string) (string, error) {
	fmt.Printf("running: %s org %s --guid\n", a.CfCliPath, name)
	bytes, err := exec.Command(a.CfCliPath, "org", name, "--guid").CombinedOutput()
	return strings.TrimSpace(string(bytes)), err
}

func (a *Adapter) AppCount(orgGuid string) (int, error) {
	fmt.Printf("running: %s curl \"/v2/apps?q=organization_guid%%20IN%%20%s\"\n", a.CfCliPath, orgGuid)
	bytes, err := exec.Command(a.CfCliPath, "curl", fmt.Sprintf("/v2/apps?q=organization_guid%%20IN%%20%s", orgGuid)).CombinedOutput()
	apps := &Apps{}
	if err := json.Unmarshal(bytes, apps); err != nil {
		return -1, err
	}
	return apps.TotalResults, err
}

func (a *Adapter) CheckApp(guid string) ([]byte, error) {
	fmt.Printf("running: %s curl \"/v2/apps/%s/summary/\"\n", a.CfCliPath, guid)
	bytes, err := exec.Command(a.CfCliPath, "curl", fmt.Sprintf("/v2/apps/%s/summary", guid)).CombinedOutput()
	return bytes, err
}

func (a *Adapter) AllowAccess(sourceApp, destApp string, port int, protocol string) error {
	portStr := fmt.Sprintf("%d", port)
	fmt.Printf("running: cf allow-access %s %s --port %s --protocol tcp\n", sourceApp, destApp, portStr)
	return exec.Command("cf", "allow-access", sourceApp, destApp, "--port", portStr, "--protocol", "tcp").Run()
}

func (a *Adapter) DenyAccess(sourceApp, destApp string, port int, protocol string) error {
	portStr := fmt.Sprintf("%d", port)
	fmt.Printf("running: cf deny-access %s %s --port %s --protocol tcp\n", sourceApp, destApp, portStr)
	return exec.Command("cf", "deny-access", sourceApp, destApp, "--port", portStr, "--protocol", "tcp").Run()
}

func (a *Adapter) CreateQuota(name, memory string, instanceMemory, routes, serviceInstances, appInstances, routePorts int) error {
	instanceMemoryStr := fmt.Sprintf("%d", instanceMemory)
	routesStr := fmt.Sprintf("%d", routes)
	serviceInstancesStr := fmt.Sprintf("%d", serviceInstances)
	appInstancesStr := fmt.Sprintf("%d", appInstances)
	routePortsStr := fmt.Sprintf("%d", routePorts)
	fmt.Printf("running cf create-quota %s -m %s -i %s -r %s -s %s -a %s --reserved-route-ports %s\n", name, memory, instanceMemoryStr, routesStr, serviceInstancesStr, appInstancesStr, routePortsStr)
	return exec.Command("cf", "create-quota", name, "-m", memory, "-i", instanceMemoryStr, "-r", routesStr, "-s", serviceInstancesStr, "-a", appInstancesStr, "--reserved-route-ports", routePortsStr).Run()
}

func (a *Adapter) SetQuota(org, quota string) error {
	fmt.Printf("running cf set-quota %s %s\n", org, quota)
	return exec.Command("cf", "set-quota", org, quota).Run()
}

func (a *Adapter) DeleteQuota(quota string) error {
	fmt.Printf("running cf delete-quota %s -f\n", quota)
	return exec.Command("cf", "delete-quota", quota, "-f").Run()
}
