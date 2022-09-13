package cni

import (
	"context"
	"fmt"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
)

//go:generate counterfeiter -o ../fakes/cni_library.go --fake-name CNILibrary . cniLibrary
type cniLibrary interface {
	libcni.CNI
}

type CNIController struct {
	CNIConfig         libcni.CNI
	NetworkConfigList *libcni.NetworkConfigList
}

func (c *CNIController) Up(namespacePath, handle string, metadata map[string]interface{}, legacyNetConf map[string]interface{}) (types.Result, error) {
	var result types.Result
	var err error

	if c.NetworkConfigList == nil {
		return result, nil
	}

	runtimeConfig := &libcni.RuntimeConf{
		ContainerID: handle,
		NetNS:       namespacePath,
		IfName:      "eth0",
	}

	extraKeys := map[string]interface{}{}
	if len(metadata) > 0 {
		extraKeys["metadata"] = metadata
	}
	if len(legacyNetConf) > 0 {
		extraKeys["runtimeConfig"] = legacyNetConf
	}

	for i, networkConfig := range c.NetworkConfigList.Plugins {
		networkConfig, err = libcni.InjectConf(networkConfig, extraKeys)
		if err != nil {
			return nil, fmt.Errorf("adding extra data to CNI config: %s", err)
		}
		c.NetworkConfigList.Plugins[i] = networkConfig
	}

	result, err = c.CNIConfig.AddNetworkList(context.TODO(), c.NetworkConfigList, runtimeConfig)
	if err != nil {
		return nil, fmt.Errorf("add network list failed: %s", err)
	}

	return result, nil
}

func (c *CNIController) Down(namespacePath, handle string) error {
	var err error

	runtimeConfig := &libcni.RuntimeConf{
		ContainerID: handle,
		NetNS:       namespacePath,
		IfName:      "eth0",
	}

	err = c.CNIConfig.DelNetworkList(context.TODO(), c.NetworkConfigList, runtimeConfig)

	if err != nil {
		return fmt.Errorf("del network failed: %s", err)
	}

	return nil
}
