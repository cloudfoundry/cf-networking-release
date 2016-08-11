package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/types"
)

//go:generate counterfeiter -o ../fakes/cniController.go --fake-name CNIController . cniController
type cniController interface {
	Up(namespacePath, handle, spec string) (*types.Result, error)
	Down(namespacePath, handle, spec string) error
}

//go:generate counterfeiter -o ../fakes/mounter.go --fake-name Mounter . mounter
type mounter interface {
	IdempotentlyMount(source, target string) error
	RemoveMount(target string) error
}

type Manager struct {
	CNIController cniController
	Mounter       mounter
	BindMountRoot string
}

type Properties struct {
	ContainerIP net.IP `json:"network.external-networker.container-ip"`
}

func (m *Manager) Up(pid int, containerHandle, encodedGardenProperties string) (*Properties, error) {
	if pid == 0 {
		return nil, errors.New("up missing pid")
	}
	if containerHandle == "" {
		return nil, errors.New("up missing container handle")
	}

	var properties map[string]string
	err := json.Unmarshal([]byte(encodedGardenProperties), &properties)
	if err != nil {
		return nil, fmt.Errorf("parsing garden properties: %s", err)
	}

	procNsPath := fmt.Sprintf("/proc/%d/ns/net", pid)
	bindMountPath := filepath.Join(m.BindMountRoot, containerHandle)

	err = m.Mounter.IdempotentlyMount(procNsPath, bindMountPath)
	if err != nil {
		return nil, fmt.Errorf("failed mounting %s to %s: %s", procNsPath, bindMountPath, err)
	}

	result, err := m.CNIController.Up(bindMountPath, containerHandle, encodedGardenProperties)
	if err != nil {
		return nil, fmt.Errorf("cni up failed: %s", err)
	}

	if result == nil {
		return nil, errors.New("cni up failed: no ip allocated")
	}

	return &Properties{
		ContainerIP: result.IP4.IP.IP,
	}, nil
}

func (m *Manager) Down(containerHandle string, encodedGardenProperties string) error {
	if containerHandle == "" {
		return errors.New("down missing container handle")
	}

	bindMountPath := filepath.Join(m.BindMountRoot, containerHandle)

	err := m.CNIController.Down(bindMountPath, containerHandle, encodedGardenProperties)
	if err != nil {
		return fmt.Errorf("cni down failed: %s", err)
	}

	err = m.Mounter.RemoveMount(bindMountPath)
	if err != nil {
		return fmt.Errorf("failed removing mount %s: %s", bindMountPath, err)
	}

	return nil
}
