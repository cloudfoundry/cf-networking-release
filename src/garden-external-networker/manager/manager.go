package manager

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
	"github.com/containernetworking/cni/pkg/types"
)

//go:generate counterfeiter -o ../fakes/cniController.go --fake-name CNIController . cniController
type cniController interface {
	Up(namespacePath, handle string, properties map[string]string) (*types.Result, error)
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
	containerIP := result.IP4.IP.IP

	if err := m.NetOutProvider.Initialize(m.Logger, containerHandle, containerIP, m.OverlayNetwork); err != nil {
		return nil, fmt.Errorf("initialize net out: %s", err)
	}

	err = m.NetInProvider.Initialize(containerHandle)
	if err != nil {
		return nil, fmt.Errorf("initialize iptables for netin: %s", err)
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
		return fmt.Errorf("cni down failed: %s", err)
	}

	err = m.Mounter.RemoveMount(bindMountPath)
	if err != nil {
		return fmt.Errorf("failed removing mount %s: %s", bindMountPath, err)
	}

	if err = m.NetOutProvider.Cleanup(containerHandle); err != nil {
		return fmt.Errorf("remove net out: %s", err)
	}

	err = m.NetInProvider.Cleanup(containerHandle)
	if err != nil {
		return fmt.Errorf("failed removing iptables for netin: %s", err)
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
		panic(err)
	}

	containerPort := inputs.ContainerPort
	if containerPort == 0 {
		containerPort = hostPort
	}

	containerIP := inputs.ContainerIP
	hostIP := inputs.HostIP

	err = m.NetInProvider.AddRule(containerHandle, hostPort, containerPort, hostIP, containerIP)
	if err != nil {
		return nil, err
	}

	return &NetInOutputs{
		HostPort:      hostPort,
		ContainerPort: containerPort,
	}, nil
}
