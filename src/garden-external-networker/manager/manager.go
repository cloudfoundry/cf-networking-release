package manager

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
)

//go:generate counterfeiter -o ../fakes/cniController.go --fake-name CNIController . cniController
type cniController interface {
	Up(namespacePath, handle string, properties map[string]string) (types.Result, error)
	Down(namespacePath, handle string) error
}

//go:generate counterfeiter -o ../fakes/mounter.go --fake-name Mounter . mounter
type mounter interface {
	IdempotentlyMount(source, target string) error
	RemoveMount(target string) error
}

//go:generate counterfeiter -o ../fakes/portAllocator.go --fake-name PortAllocator . portAllocator
type portAllocator interface {
	AllocatePort(handle string, port int) (int, error)
	ReleaseAllPorts(handle string) error
}

//go:generate counterfeiter -o ../fakes/netin_provider.go --fake-name NetInProvider . netInProvider
type netInProvider interface {
	Initialize(containerHandle string) error
	Cleanup(containerHandle string) error
	AddRule(containerHandle string, hostPort, containerPort int, hostIP, containerIP string) error
}

//go:generate counterfeiter -o ../fakes/netout_provider.go --fake-name NetOutProvider . netOutProvider
type netOutProvider interface {
	Initialize(logger lager.Logger, containerHandle string, containerIP net.IP, overlayNetwork string) error
	Cleanup(containerHandle string) error
	InsertRule(containerHandle string, rule garden.NetOutRule, containerIP string) error
	BulkInsertRules(containerHandle string, rules []garden.NetOutRule, containerIP string) error
}

type Manager struct {
	Logger         lager.Logger
	CNIController  cniController
	Mounter        mounter
	BindMountRoot  string
	OverlayNetwork string
	PortAllocator  portAllocator
	NetInProvider  netInProvider
	NetOutProvider netOutProvider
}

type UpInputs struct {
	Pid        int
	Properties map[string]string
	NetOut     []garden.NetOutRule `json:"netout_rules"`
}
type UpOutputs struct {
	Properties struct {
		ContainerIP      string `json:"garden.network.container-ip"`
		DeprecatedHostIP string `json:"garden.network.host-ip"`
	} `json:"properties"`
}

func (m *Manager) Up(containerHandle string, inputs UpInputs) (*UpOutputs, error) {
	if inputs.Pid == 0 {
		return nil, errors.New("up missing pid")
	}
	if containerHandle == "" {
		return nil, errors.New("up missing container handle")
	}

	procNsPath := fmt.Sprintf("/proc/%d/ns/net", inputs.Pid)
	bindMountPath := filepath.Join(m.BindMountRoot, containerHandle)

	err := m.Mounter.IdempotentlyMount(procNsPath, bindMountPath)
	if err != nil {
		return nil, fmt.Errorf("failed mounting %s to %s: %s", procNsPath, bindMountPath, err)
	}

	result, err := m.CNIController.Up(bindMountPath, containerHandle, inputs.Properties)
	if err != nil {
		return nil, fmt.Errorf("cni up failed: %s", err)
	}

	if result == nil {
		return nil, errors.New("cni up failed: no ip allocated")
	}

	result020, err := result.GetAsVersion("0.2.0")
	if err != nil {
		return nil, fmt.Errorf("cni plugin result version incompatible: %s", err) // not tested
	}

	containerIP := result020.(*types020.Result).IP4.IP.IP

	if err := m.NetOutProvider.Initialize(m.Logger, containerHandle, containerIP, m.OverlayNetwork); err != nil {
		return nil, fmt.Errorf("initialize net out: %s", err)
	}

	err = m.NetInProvider.Initialize(containerHandle)
	if err != nil {
		return nil, fmt.Errorf("initialize iptables for netin: %s", err)
	}

	if err := m.NetOutProvider.BulkInsertRules(containerHandle, inputs.NetOut, containerIP.String()); err != nil {
		return nil, fmt.Errorf("bulk insert: %s", err)
	}

	outputs := UpOutputs{}
	outputs.Properties.ContainerIP = containerIP.String()
	outputs.Properties.DeprecatedHostIP = "255.255.255.255"
	return &outputs, nil
}

func (m *Manager) Down(containerHandle string) error {
	if containerHandle == "" {
		return errors.New("down missing container handle")
	}

	bindMountPath := filepath.Join(m.BindMountRoot, containerHandle)

	err := m.CNIController.Down(bindMountPath, containerHandle)
	if err != nil {
		return fmt.Errorf("cni down: %s", err)
	}

	if err = m.Mounter.RemoveMount(bindMountPath); err != nil {
		m.Logger.Error("removing mount", err, lager.Data{"bind mount path": bindMountPath})
	}

	if err = m.NetOutProvider.Cleanup(containerHandle); err != nil {
		m.Logger.Error("net out cleanup", err)
	}

	if err = m.NetInProvider.Cleanup(containerHandle); err != nil {
		m.Logger.Error("net in cleanup", err)
	}

	if err = m.PortAllocator.ReleaseAllPorts(containerHandle); err != nil {
		m.Logger.Error("releasing ports", err)
	}

	return nil
}

type NetOutInputs struct {
	ContainerIP string            `json:"container_ip"`
	NetOutRule  garden.NetOutRule `json:"netout_rule"`
}

func (m *Manager) NetOut(containerHandle string, inputs NetOutInputs) error {
	return m.NetOutProvider.InsertRule(containerHandle, inputs.NetOutRule, inputs.ContainerIP)
}

type NetInInputs struct {
	HostIP        string
	HostPort      int
	ContainerIP   string
	ContainerPort int
}

type NetInOutputs struct {
	HostPort      int `json:"host_port"`
	ContainerPort int `json:"container_port"`
}

func (m *Manager) NetIn(containerHandle string, inputs NetInInputs) (*NetInOutputs, error) {
	hostPort, err := m.PortAllocator.AllocatePort(containerHandle, inputs.HostPort)
	if err != nil {
		return nil, fmt.Errorf("allocate port: %s", err)
	}

	containerPort := inputs.ContainerPort
	if containerPort == 0 {
		containerPort = hostPort
	}

	containerIP := inputs.ContainerIP
	hostIP := inputs.HostIP

	err = m.NetInProvider.AddRule(containerHandle, hostPort, containerPort, hostIP, containerIP)
	if err != nil {
		return nil, fmt.Errorf("add rule: %s", err)
	}

	return &NetInOutputs{
		HostPort:      hostPort,
		ContainerPort: containerPort,
	}, nil
}

type BulkNetOutInputs struct {
	ContainerIP string              `json:"container_ip"`
	NetOutRules []garden.NetOutRule `json:"netout_rules"`
}

func (m *Manager) BulkNetOut(containerHandle string, inputs BulkNetOutInputs) error {
	err := m.NetOutProvider.BulkInsertRules(containerHandle, inputs.NetOutRules, inputs.ContainerIP)
	if err != nil {
		return fmt.Errorf("insert rule: %s", err)
	}
	return nil
}
