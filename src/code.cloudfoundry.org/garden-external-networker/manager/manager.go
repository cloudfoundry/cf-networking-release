package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"code.cloudfoundry.org/garden"
	"github.com/containernetworking/cni/pkg/types"
	types040 "github.com/containernetworking/cni/pkg/types/040"
)

//go:generate counterfeiter -o ../fakes/proxyRedirect.go --fake-name ProxyRedirect . proxyRedirect
type proxyRedirect interface {
	Apply(containerNamespace string) error
}

//go:generate counterfeiter -o ../fakes/cniController.go --fake-name CNIController . cniController
type cniController interface {
	Up(namespacePath, handle string, metadata map[string]interface{}, legacyNetConf map[string]interface{}) (types.Result, error)
	Down(namespacePath, handle string) error
}

//go:generate counterfeiter -o ../fakes/mounter.go --fake-name Mounter . mounter
type mounter interface {
	IdempotentlyMount(source, target string) error
	RemoveMount(target string) error
}

//go:generate counterfeiter -o ../fakes/portAllocator.go --fake-name PortAllocator . portAllocator
type portAllocator interface {
	AllocatePort(handle string, port uint32) (uint32, error)
	ReleaseAllPorts(handle string) error
}

type Manager struct {
	Logger        io.Writer
	CNIController cniController
	Mounter       mounter
	BindMountRoot string
	PortAllocator portAllocator
	SearchDomains []string
	ProxyRedirect proxyRedirect
}

type UpInputs struct {
	Pid        int
	Properties map[string]interface{}
	NetOut     []garden.NetOutRule `json:"netout_rules"`
	NetIn      []garden.NetIn      `json:"netin"`
}
type UpOutputs struct {
	Properties struct {
		ContainerIP      string `json:"garden.network.container-ip"`
		DeprecatedHostIP string `json:"garden.network.host-ip"`
		MappedPorts      string `json:"garden.network.mapped-ports"`
		Interface        string `json:"garden.network.interface,omitempty"`
	} `json:"properties"`
	DNSServers    []string `json:"dns_servers,omitempty"`
	SearchDomains []string `json:"search_domains,omitempty"`
}

func (m *Manager) Up(containerHandle string, inputs UpInputs) (*UpOutputs, error) {
	if containerHandle == "" {
		return nil, errors.New("up missing container handle")
	}

	procNsPath := fmt.Sprintf("/proc/%d/ns/net", inputs.Pid)
	if inputs.Pid == 0 {
		procNsPath = "/proc/self/fd/3"
	}

	bindMountPath := filepath.Join(m.BindMountRoot, containerHandle)

	err := m.Mounter.IdempotentlyMount(procNsPath, bindMountPath)
	if err != nil {
		return nil, fmt.Errorf("failed mounting %s to %s: %s", procNsPath, bindMountPath, err)
	}

	mappedPorts := []garden.PortMapping{}
	for i := range inputs.NetIn {
		if inputs.NetIn[i].HostPort == 0 {
			hostPort, err := m.PortAllocator.AllocatePort(containerHandle, inputs.NetIn[i].HostPort)
			if err != nil {
				return nil, fmt.Errorf("allocating port: %s", err)
			}
			inputs.NetIn[i].HostPort = hostPort
		}

		mappedPorts = append(mappedPorts, garden.PortMapping{
			HostPort:      inputs.NetIn[i].HostPort,
			ContainerPort: inputs.NetIn[i].ContainerPort,
		})
	}

	result, err := m.CNIController.Up(
		bindMountPath,
		containerHandle,
		inputs.Properties,
		map[string]interface{}{
			"portMappings": inputs.NetIn,
			"netOutRules":  inputs.NetOut,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("cni up failed: %s", err)
	}

	if result == nil {
		return nil, errors.New("cni up failed: no ip allocated")
	}

	result040, err := result.GetAsVersion("0.4.0")
	if err != nil {
		return nil, fmt.Errorf("cni plugin result version incompatible: %s", err) // not tested
	}

	err = m.ProxyRedirect.Apply(bindMountPath)
	if err != nil {
		return nil, fmt.Errorf("proxy redirect apply: %s", err)
	}

	assertedResult := result040.(*types040.Result)

	var containerIP *types040.IPConfig
	for _, ip := range assertedResult.IPs {
		if ip.Version == "4" {
			containerIP = ip
			break
		}
	}

	if containerIP == nil {
		return nil, errors.New("expected an IPv4 address in the CNI result")
	}

	// support CNI version lower than 0.4.0 which don't carry the Interface name in the result
	interfaceName := ""
	if containerIP.Interface != nil {
		if interfacesLen := len(assertedResult.Interfaces); *containerIP.Interface >= interfacesLen {
			return nil, fmt.Errorf("no corresponding interface found, interface index: %d, number of interfaces: %d", *containerIP.Interface, interfacesLen)
		}
		interfaceName = assertedResult.Interfaces[*containerIP.Interface].Name
	}

	outputs := UpOutputs{}
	outputs.Properties.MappedPorts = toJson(mappedPorts)
	outputs.Properties.ContainerIP = containerIP.Address.IP.String()
	outputs.Properties.Interface = interfaceName
	outputs.Properties.DeprecatedHostIP = "255.255.255.255"
	outputs.DNSServers = assertedResult.DNS.Nameservers
	outputs.SearchDomains = m.SearchDomains
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
		fmt.Fprintf(m.Logger, "removing bind mount %s: %s\n", bindMountPath, err)
	}

	if err = m.PortAllocator.ReleaseAllPorts(containerHandle); err != nil {
		fmt.Fprintf(m.Logger, "releasing ports: %s\n", err)
	}

	return nil
}

func toJson(mappedPorts []garden.PortMapping) string {
	bytes, err := json.Marshal(mappedPorts)
	if err != nil {
		panic(err) // untested, should never happen
	}

	return string(bytes)
}
