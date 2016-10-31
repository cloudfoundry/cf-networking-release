package cni

import (
	"fmt"

	"code.cloudfoundry.org/lager"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
)

//go:generate counterfeiter -o ../fakes/cni_library.go --fake-name CNILibrary . cniLibrary
type cniLibrary interface {
	libcni.CNI
}

type CNIController struct {
	Logger lager.Logger

	CNIConfig      libcni.CNI
	NetworkConfigs []*libcni.NetworkConfig
}

func (c *CNIController) Up(namespacePath, handle string, properties map[string]string) (*types.Result, error) {
	var result *types.Result
	var err error

	for i, networkConfig := range c.NetworkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		if len(properties) > 0 {
			networkConfig, err = libcni.InjectConf(networkConfig, "metadata", properties)
			if err != nil {
				return nil, fmt.Errorf("adding garden properties to CNI config: %s", err)
			}
		}

		c.Logger.Info("up-add-network-start", lager.Data{"networkConfig": string(networkConfig.Bytes), "runtimeConfig": runtimeConfig})
		result, err = c.CNIConfig.AddNetwork(networkConfig, runtimeConfig)
		if err != nil {
			return nil, fmt.Errorf("add network failed: %s", err)
		}
		c.Logger.Info("up-add-network-result", lager.Data{"name": networkConfig.Network.Name, "type": networkConfig.Network.Type, "result": result.String()})
	}
	c.Logger.Info("up-complete", lager.Data{"numConfigs": len(c.NetworkConfigs)})

	return result, nil
}

func (c *CNIController) Down(namespacePath, handle string) error {
	var err error
	for i, networkConfig := range c.NetworkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		c.Logger.Info("down-del-network-start", lager.Data{"networkConfig": string(networkConfig.Bytes), "runtimeConfig": runtimeConfig})
		err = c.CNIConfig.DelNetwork(networkConfig, runtimeConfig)
		if err != nil {
			return fmt.Errorf("del network failed: %s", err)
		}

		c.Logger.Info("down-del-network-result", lager.Data{"name": networkConfig.Network.Name, "type": networkConfig.Network.Type})
	}
	c.Logger.Info("down-complete", lager.Data{"numConfigs": len(c.NetworkConfigs)})

	return nil
}
