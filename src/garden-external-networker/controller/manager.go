package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"lib/rules"
	"net"
	"path/filepath"
	"strconv"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
	"github.com/containernetworking/cni/pkg/types"
)

//go:generate counterfeiter -o ../fakes/cniController.go --fake-name CNIController . cniController
type cniController interface {
	Up(namespacePath, handle string, properties map[string]string) (*types.Result, error)
	Down(namespacePath, handle string, properties map[string]string) error
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

type Manager struct {
	Logger         lager.Logger
	CNIController  cniController
	Mounter        mounter
	BindMountRoot  string
	IPTables       rules.IPTables
	OverlayNetwork string
	PortAllocator  portAllocator
}

type UpResultProperties struct {
	ContainerIP      net.IP `json:"garden.network.container-ip"`
	DeprecatedHostIP net.IP `json:"garden.network.host-ip"`
}

func ExtractGardenProperties(encodedGardenProperties string) (map[string]string, error) {
	if encodedGardenProperties == "" {
		return nil, nil
	}
	props := make(map[string]string)
	err := json.Unmarshal([]byte(encodedGardenProperties), &props)
	if err != nil {
		return nil, fmt.Errorf("unmarshal garden properties: %s", err)
	}
	return props, nil
}

func (m *Manager) Up(pid int, containerHandle, encodedGardenProperties string) (*UpResultProperties, error) {
	if pid == 0 {
		return nil, errors.New("up missing pid")
	}
	if containerHandle == "" {
		return nil, errors.New("up missing container handle")
	}

	gardenProps, err := ExtractGardenProperties(encodedGardenProperties)
	if err != nil {
		return nil, err
	}

	procNsPath := fmt.Sprintf("/proc/%d/ns/net", pid)
	bindMountPath := filepath.Join(m.BindMountRoot, containerHandle)

	err = m.Mounter.IdempotentlyMount(procNsPath, bindMountPath)
	if err != nil {
		return nil, fmt.Errorf("failed mounting %s to %s: %s", procNsPath, bindMountPath, err)
	}

	result, err := m.CNIController.Up(bindMountPath, containerHandle, gardenProps)
	if err != nil {
		return nil, fmt.Errorf("cni up failed: %s", err)
	}

	if result == nil {
		return nil, errors.New("cni up failed: no ip allocated")
	}

	if err := m.InitializeIPTablesNetOut(containerHandle, result.IP4.IP.IP); err != nil {
		return nil, fmt.Errorf("initialize net out: %s", err)
	}

	return &UpResultProperties{
		ContainerIP:      result.IP4.IP.IP,
		DeprecatedHostIP: net.ParseIP("255.255.255.255"),
	}, nil
}

func (m *Manager) Down(containerHandle string, encodedGardenProperties string) error {
	if containerHandle == "" {
		return errors.New("down missing container handle")
	}

	gardenProps, err := ExtractGardenProperties(encodedGardenProperties)
	if err != nil {
		return err
	}

	bindMountPath := filepath.Join(m.BindMountRoot, containerHandle)

	err = m.CNIController.Down(bindMountPath, containerHandle, gardenProps)
	if err != nil {
		return fmt.Errorf("cni down failed: %s", err)
	}

	err = m.Mounter.RemoveMount(bindMountPath)
	if err != nil {
		return fmt.Errorf("failed removing mount %s: %s", bindMountPath, err)
	}

	if err = m.RemoveIPTablesNetOut(containerHandle); err != nil {
		return fmt.Errorf("remove net out: %s", err)
	}

	return nil
}

type NetOutProperties struct {
	ContainerIP string            `json:"container_ip"`
	NetOutRule  garden.NetOutRule `json:"netout_rule"`
}

func (m *Manager) NetOut(containerHandle string, encodedGardenProperties string) error {
	var properties NetOutProperties
	err := json.Unmarshal([]byte(encodedGardenProperties), &properties)
	if err != nil {
		return fmt.Errorf("unmarshaling net-out properties: %s", err)
	}
	m.Logger.Info("net-out", lager.Data{"properties": properties})

	chain := fmt.Sprintf("netout--%s", containerHandle)
	if len(chain) > 28 {
		chain = chain[:28]
	}

	rule := properties.NetOutRule
	for _, network := range rule.Networks {
		if len(rule.Ports) > 0 && udpOrTcp(rule.Protocol) {
			for _, portRange := range properties.NetOutRule.Ports {
				ruleSpec := rules.NewNetOutWithPortsRule(
					properties.ContainerIP,
					network.Start.String(),
					network.End.String(),
					int(portRange.Start),
					int(portRange.End),
					lookupProtocol(properties.NetOutRule.Protocol),
				)
				err = m.IPTables.Insert("filter", chain, 1, ruleSpec.Properties...)
				if err != nil {
					return fmt.Errorf("inserting net-out rule: %s", err)
				}
			}
		} else {
			ruleSpec := rules.NewNetOutRule(
				properties.ContainerIP,
				network.Start.String(),
				network.End.String(),
			)
			err = m.IPTables.Insert("filter", chain, 1, ruleSpec.Properties...)
			if err != nil {
				return fmt.Errorf("inserting net-out rule: %s", err)
			}
		}
	}

	return nil
}
func udpOrTcp(protocol garden.Protocol) bool {
	return protocol == garden.ProtocolTCP || protocol == garden.ProtocolUDP
}

func lookupProtocol(protocol garden.Protocol) string {
	switch protocol {
	case garden.ProtocolTCP:
		return "tcp"
	case garden.ProtocolUDP:
		return "udp"
	default:
		return "all"
	}
}

func (m *Manager) RemoveIPTablesNetOut(containerHandle string) error {
	chain := fmt.Sprintf("netout--%s", containerHandle)
	if len(chain) > 28 {
		chain = chain[:28]
	}

	err := m.IPTables.Delete("filter", "FORWARD", []string{"--jump", chain}...)
	if err != nil {
		return fmt.Errorf("deleting rule: %s", err)
	}

	err = m.IPTables.ClearChain("filter", chain)
	if err != nil {
		return fmt.Errorf("creating chain: %s", err)
	}

	err = m.IPTables.DeleteChain("filter", chain)
	if err != nil {
		return fmt.Errorf("creating chain: %s", err)
	}

	return nil
}

func (m *Manager) InitializeIPTablesNetOut(containerHandle string, containerIP net.IP) error {
	chain := fmt.Sprintf("netout--%s", containerHandle)
	if len(chain) > 28 {
		chain = chain[:28]
	}
	err := m.IPTables.NewChain("filter", chain)
	if err != nil {
		return fmt.Errorf("creating chain: %s", err)
	}

	err = m.IPTables.Insert("filter", "FORWARD", 1, []string{"--jump", chain}...)
	if err != nil {
		return fmt.Errorf("inserting rule: %s", err)
	}

	ruleSpecs := []rules.Rule{
		rules.NewNetOutRelatedEstablishedRule(containerIP.String(), m.OverlayNetwork),
		rules.NewNetOutDefaultRejectRule(containerIP.String(), m.OverlayNetwork),
	}

	for _, spec := range ruleSpecs {
		err = spec.Enforce("filter", chain, m.IPTables, m.Logger)
		if err != nil {
			return err
		}
	}

	return nil
}

type NetInProperties struct {
	HostIP        string
	HostPort      int `json:"host_port"`
	ContainerIP   string
	ContainerPort int `json:"container_port"`
	GroupID       string
}

func (m *Manager) NetIn(containerHandle, encodedGardenProperties string) (*NetInProperties, error) {
	gardenProps, err := ExtractGardenProperties(encodedGardenProperties)
	if err != nil {
		panic(err)
	}

	hostPort, err := strconv.Atoi(gardenProps["host-port"])
	if err != nil {
		panic(err)
	}

	hostPort, err = m.PortAllocator.AllocatePort(containerHandle, hostPort)
	if err != nil {
		panic(err)
	}

	containerPort, err := strconv.Atoi(gardenProps["container-port"])
	if err != nil {
		panic(err)
	}

	if containerPort == 0 {
		containerPort = hostPort
	}

	containerIP := gardenProps["container-ip"]
	hostIP := gardenProps["host-ip"]
	groupID := gardenProps["app_id"]

	chainName := fmt.Sprintf("netin--%s", containerHandle)
	if len(chainName) > 29 {
		chainName = chainName[:29]
	}

	rule := rules.NewNetInRule(containerIP, containerPort, hostIP, hostPort, groupID)
	err = rule.Enforce("nat", chainName, m.IPTables, m.Logger)
	if err != nil {
		panic(err)
	}

	return &NetInProperties{
		HostIP:        hostIP,
		HostPort:      hostPort,
		ContainerIP:   containerIP,
		ContainerPort: containerPort,
		GroupID:       groupID,
	}, nil
}
