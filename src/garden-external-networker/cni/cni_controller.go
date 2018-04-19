package cni

import (
	"fmt"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
)

//go:generate counterfeiter -o ../fakes/cni_library.go --fake-name CNILibrary . cniLibrary
type cniLibrary interface {
	libcni.CNI
}

type CNIController struct {
	CNIConfig          libcni.CNI
	NetworkConfigLists []*libcni.NetworkConfigList
}

func (c *CNIController) Up(namespacePath, handle string, metadata map[string]interface{}, legacyNetConf map[string]interface{}) (types.Result, error) {
	var result types.Result
	var err error

	for i, networkConfigList := range c.NetworkConfigLists {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		extraKeys := map[string]interface{}{}
		if len(metadata) > 0 {
			extraKeys["metadata"] = metadata
		}
		if len(legacyNetConf) > 0 {
			extraKeys["runtimeConfig"] = legacyNetConf
		}

		for i, networkConfig := range networkConfigList.Plugins {
			networkConfig, err = libcni.InjectConf(networkConfig, extraKeys)
			if err != nil {
				return nil, fmt.Errorf("adding extra data to CNI config: %s", err)
			}
			networkConfigList.Plugins[i] = networkConfig
		}

		result, err = c.CNIConfig.AddNetworkList(networkConfigList, runtimeConfig)
		if err != nil {
			return nil, fmt.Errorf("add network list failed: %s", err)
		}
	}

	return result, nil
}

func (c *CNIController) Down(namespacePath, handle string) error {
	var err error
	for i, networkConfigList := range c.NetworkConfigLists {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		err = c.CNIConfig.DelNetworkList(networkConfigList, runtimeConfig)
		if err != nil {
			return fmt.Errorf("del network failed: %s", err)
		}
	}

	return nil
}
