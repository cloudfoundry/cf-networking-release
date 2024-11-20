package cf_cli_adapter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Adapter struct {
	cfCliPath    string
	majorVersion int
	cfHomePath   string
	logWriter    io.Writer
}

func NewAdapter() *Adapter {
	// cf version 6.51.0+2acd15650.2020-04-07
	// cf7 version 7.0.2+17b4eeafd.2020-07-24
	bytes, err := exec.Command("cf", "version").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", bytes)
		panic(err)
	}
	versionString := string(bytes)
	versionString = strings.Split(versionString, " ")[2]
	versionString = strings.Split(versionString, ".")[0]
	majorVersion, err := strconv.Atoi(versionString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", bytes)
		panic(err)
	}

	return &Adapter{cfCliPath: "cf", majorVersion: majorVersion, logWriter: os.Stdout}
}

func NewAdapterWithHome(home string) *Adapter {
	adapter := NewAdapter()
	adapter.cfHomePath = home

	return adapter
}

func NewAdapterWithLogWriter(logWriter io.Writer) *Adapter {
	adapter := NewAdapter()
	adapter.logWriter = logWriter

	return adapter
}

func (a *Adapter) CfCliV6() bool {
	return a.majorVersion < 7
}

func (a *Adapter) LogCommand(commandArgs []string) {
	fmt.Fprintf(a.logWriter, "running: %s %v\n", a.cfCliPath, commandArgs)
}

func (a *Adapter) CreateOrg(name string) error {
	commandArgs := []string{"create-org", name}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) DeleteOrg(name string) error {
	commandArgs := []string{"delete-org", "-f", name}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) CreateSpace(spaceName, orgName string) error {
	commandArgs := []string{"create-space", spaceName, "-o", orgName}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) TargetOrg(name string) error {
	commandArgs := []string{"target", "-o", name}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) TargetSpace(name string) error {
	commandArgs := []string{"target", "-s", name}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) SetApiWithSsl(api string) error {
	commandArgs := []string{"api", api}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) SetApiWithoutSsl(api string) error {
	commandArgs := []string{"api", api, "--skip-ssl-validation"}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) Auth(user, password string) error {
	a.LogCommand([]string{"auth", "<user>", "<pass>"})
	cmd := exec.Command(a.cfCliPath, "auth", user, password)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) Push(name, directory, manifestFile string) error {
	commandArgs := []string{"push", name, "-p", directory, "-f", manifestFile}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	bytes, err := a.runCombinedOutput(cmd)
	fmt.Fprintf(a.logWriter, "output: %s\n", string(bytes))
	return err
}

func (a *Adapter) Delete(appName string) error {
	commandArgs := []string{"delete", "-f", appName}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) Scale(name string, instances int) error {
	instancesStr := fmt.Sprintf("%d", instances)
	commandArgs := []string{"scale", name, "-i", instancesStr}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) AppGuid(name string) (string, error) {
	commandArgs := []string{"app", name, "--guid"}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	bytes, err := a.runCombinedOutput(cmd)
	return strings.TrimSpace(string(bytes)), err
}

func (a *Adapter) SpaceGuid(name string) (string, error) {
	commandArgs := []string{"space", name, "--guid"}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	bytes, err := a.runCombinedOutput(cmd)
	return strings.TrimSpace(string(bytes)), err
}

type Apps struct {
	TotalResults int `json:"total_results"`
}

func (a *Adapter) OrgGuid(name string) (string, error) {
	commandArgs := []string{"org", name, "--guid"}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	bytes, err := a.runCombinedOutput(cmd)
	return strings.TrimSpace(string(bytes)), err
}

func (a *Adapter) Curl(method, path, inputFile string) ([]byte, error) {
	var commandArgs []string
	if inputFile != "" {
		commandArgs = []string{"curl", "-X", method, "-d", fmt.Sprintf("@%s", inputFile), path}
		a.LogCommand(commandArgs)
		cmd := exec.Command(a.cfCliPath, commandArgs...)
		return a.runCombinedOutput(cmd)
	}

	commandArgs = []string{"curl", "-X", method, path}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCombinedOutput(cmd)
}

func (a *Adapter) AppCount(orgGuid string) (int, error) {
	commandArgs := []string{"curl", fmt.Sprintf("/v2/apps?q=organization_guid%%20IN%%20%s", orgGuid)}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	bytes, err := a.runCombinedOutput(cmd)
	apps := &Apps{}
	if err := json.Unmarshal(bytes, apps); err != nil {
		return -1, err
	}
	return apps.TotalResults, err
}

func (a *Adapter) CheckApp(guid string) ([]byte, error) {
	commandArgs := []string{"curl", fmt.Sprintf("/v2/apps/%s/summary", guid)}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCombinedOutput(cmd)
}

func (a *Adapter) AddNetworkPolicy(sourceApp, destApp string, port int, protocol string) error {
	portStr := fmt.Sprintf("%d-%d", port, port)
	var commandArgs []string
	if a.CfCliV6() {
		commandArgs = []string{"add-network-policy", sourceApp, "--destination-app", destApp, "--port", portStr, "--protocol", "tcp"}
	} else {
		commandArgs = []string{"add-network-policy", sourceApp, destApp, "--port", portStr, "--protocol", "tcp"}
	}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) RemoveNetworkPolicy(sourceApp, destApp string, port int, protocol string) error {
	portStr := fmt.Sprintf("%d-%d", port, port)
	var commandArgs []string
	if a.CfCliV6() {
		commandArgs = []string{"remove-network-policy", sourceApp, "--destination-app", destApp, "--port", portStr, "--protocol", "tcp"}
	} else {
		commandArgs = []string{"remove-network-policy", sourceApp, destApp, "--port", portStr, "--protocol", "tcp"}
	}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) CleanupStaleNetworkPolicies() ([]byte, error) {
	return a.Curl("POST", "/networking/v0/external/policies/cleanup", "")
}

func (a *Adapter) CreateQuota(name, memory string, instanceMemory, routes, serviceInstances, appInstances, routePorts int) error {
	instanceMemoryStr := fmt.Sprintf("%d", instanceMemory)
	routesStr := fmt.Sprintf("%d", routes)
	serviceInstancesStr := fmt.Sprintf("%d", serviceInstances)
	appInstancesStr := fmt.Sprintf("%d", appInstances)
	routePortsStr := fmt.Sprintf("%d", routePorts)
	var commandArgs []string
	if a.CfCliV6() {
		commandArgs = []string{"create-quota", name, "-m", memory, "-i", instanceMemoryStr, "-r", routesStr, "-s", serviceInstancesStr, "-a", appInstancesStr, "--reserved-route-ports", routePortsStr}
	} else {
		commandArgs = []string{"create-org-quota", name, "-m", memory, "-i", instanceMemoryStr, "-r", routesStr, "-s", serviceInstancesStr, "-a", appInstancesStr, "--reserved-route-ports", routePortsStr}
	}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) SetQuota(org, quota string) error {
	var commandArgs []string
	if a.CfCliV6() {
		commandArgs = []string{"set-quota", org, quota}
	} else {
		commandArgs = []string{"set-org-quota", org, quota}
	}

	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) CreateSecurityGroup(name, filepath string) error {
	commandArgs := []string{"create-security-group", name, filepath}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

type ASG struct {
	Resources []struct {
		Entity struct {
			Rules []struct {
				Destination string `json:"destination"`
				Ports       string `json:"ports"`
				Protocol    string `json:"protocol"`
			} `json:"rules"`
		} `json:"entity"`
	} `json:"resources"`
}

func (a *Adapter) SecurityGroup(name string) (string, error) {
	commandArgs := []string{"curl", fmt.Sprintf("/v2/security_groups?q=name%%3A%s", name)}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	bytes, _ := a.runCombinedOutput(cmd)
	asg := &ASG{}
	if err := json.Unmarshal(bytes, asg); err != nil {
		return "", err
	}
	if len(asg.Resources) == 0 {
		return "", errors.New("no asgs with the name " + name)
	}
	rules, err := json.Marshal(asg.Resources[0].Entity.Rules)
	if err != nil {
		return "", err
	}
	return string(rules), err
}

func (a *Adapter) BindSecurityGroup(name, org, space string) error {
	var commandArgs []string
	if a.CfCliV6() {
		commandArgs = []string{"bind-security-group", name, org, space}
	} else {
		commandArgs = []string{"bind-security-group", name, org, "--space", space}
	}

	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) BindGlobalRunningSecurityGroup(name string) error {
	var commandArgs []string
	if a.CfCliV6() {
		commandArgs = []string{"bind-running-security-group", name}
	} else {
		commandArgs = []string{"bind-running-security-group", name}
	}

	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) UnbindSecurityGroup(name, org, space string) error {
	commandArgs := []string{"unbind-security-group", name, org, space}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) DeleteSecurityGroup(name string) error {
	commandArgs := []string{"delete-security-group", "-f", name}
	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) DeleteQuota(quota string) error {
	var commandArgs []string
	if a.CfCliV6() {
		commandArgs = []string{"delete-quota", quota}
	} else {
		commandArgs = []string{"delete-org-quota", quota}
	}

	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

func (a *Adapter) RunTask(appName, commandToRun string) error {
	var commandArgs []string
	if a.CfCliV6() {
		commandArgs = []string{"run-task", appName, commandToRun}
	} else {
		commandArgs = []string{"run-task", appName, "-c", commandToRun}
	}

	a.LogCommand(commandArgs)
	cmd := exec.Command(a.cfCliPath, commandArgs...)
	return a.runCommandWithTimeout(cmd)
}

type CmdErr struct {
	Out     string
	Err     string
	Message string
}

func (e *CmdErr) Error() string {
	return fmt.Sprintf("%s:\n\nOut:\n%s\n\nErr:%s\n", e.Message, e.Out, e.Err)
}

func (a Adapter) runCombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	if a.cfHomePath != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("CF_HOME=%s", a.cfHomePath))
	}

	return cmd.CombinedOutput()
}

func (a Adapter) runCommandWithTimeout(cmd *exec.Cmd) error {
	outBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}
	wrapErr := func(msg string) error {
		return &CmdErr{
			Out:     outBuffer.String(),
			Err:     errBuffer.String(),
			Message: msg,
		}
	}
	if a.cfHomePath != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("CF_HOME=%s", a.cfHomePath))
	}
	cmd.Stdout = outBuffer
	cmd.Stderr = errBuffer
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(2 * time.Minute):
		if err := cmd.Process.Kill(); err != nil {
			return wrapErr(fmt.Sprintf("command timed out and could not be killed: %s", err))
		}
		return wrapErr("command timed out")

	case err := <-done:
		if err != nil {
			return wrapErr(err.Error())
		}
	}
	return nil
}
