package cni

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/lager"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
)

type CNILoader struct {
	PluginDir string
	ConfigDir string
	Logger    lager.Logger
}

func (l *CNILoader) GetCNIConfig() *libcni.CNIConfig {
	return &libcni.CNIConfig{Path: []string{l.PluginDir}}
}

func (l *CNILoader) GetNetworkConfigs() ([]*libcni.NetworkConfig, error) {
	networkConfigs := []*libcni.NetworkConfig{}
	err := filepath.Walk(l.ConfigDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".conf") {
			return nil
		}

		conf, err := libcni.ConfFromFile(path)
		if err != nil {
			return fmt.Errorf("unable to load config from %s: %s", path, err)
		}

		networkConfigs = append(networkConfigs, conf)

		l.Logger.Info("loaded-config", lager.Data{"network": conf.Network, "raw": string(conf.Bytes)})
		return nil
	})

	if err != nil {
		return networkConfigs, fmt.Errorf("error loading config: %s", err)
	}

	return networkConfigs, nil
}

type CNIController struct {
	Logger lager.Logger

	CNIConfig      *libcni.CNIConfig
	NetworkConfigs []*libcni.NetworkConfig
}

func injectConf(original *libcni.NetworkConfig, key string, newValue interface{}) (*libcni.NetworkConfig, error) {
	config := make(map[string]interface{})
	err := json.Unmarshal(original.Bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal existing network bytes: %s", err)
	}

	if key == "" {
		return nil, fmt.Errorf("key value can not be empty")
	}

	if newValue == nil {
		return nil, fmt.Errorf("newValue must be specified")
	}

	config[key] = newValue

	newBytes, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	return libcni.ConfFromBytes(newBytes)
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
			networkConfig, err = injectConf(networkConfig, "network", map[string]interface{}{
				"properties": properties,
			})
			if err != nil {
				return nil, fmt.Errorf("adding garden properties to CNI config: %s", err)
			}
		}

		result, err = c.CNIConfig.AddNetwork(networkConfig, runtimeConfig)
		if err != nil {
			return nil, fmt.Errorf("add network failed: %s", err)
		}
		c.Logger.Info("up-result", lager.Data{"name": networkConfig.Network.Name, "type": networkConfig.Network.Type, "result": result.String()})
	}
	c.Logger.Info("up-complete", lager.Data{"numConfigs": len(c.NetworkConfigs)})

	return result, nil
}

func (c *CNIController) Down(namespacePath, handle string, properties map[string]string) error {
	var err error
	for i, networkConfig := range c.NetworkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		networkConfig, err = injectConf(networkConfig, "network", map[string]interface{}{
			"properties": properties,
		})
		if err != nil {
			return fmt.Errorf("adding garden properties to CNI config: %s", err)
		}

		err = c.CNIConfig.DelNetwork(networkConfig, runtimeConfig)
		if err != nil {
			return fmt.Errorf("del network failed: %s", err)
		}

		c.Logger.Info("down-result", lager.Data{"name": networkConfig.Network.Name, "type": networkConfig.Network.Type})
	}
	c.Logger.Info("down-complete", lager.Data{"numConfigs": len(c.NetworkConfigs)})

	return nil
}
