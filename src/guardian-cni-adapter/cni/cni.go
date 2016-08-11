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

func InjectGardenProperties(existingNetConfig *libcni.NetworkConfig, encodedGardenProperties string) (*libcni.NetworkConfig, error) {
	if encodedGardenProperties == "" {
		return existingNetConfig, nil
	}

	gardenProps, err := ExtractGardenProperties(encodedGardenProperties)
	if err != nil {
		return nil, err
	}

	if len(gardenProps) == 0 {
		return existingNetConfig, nil
	}

	return InjectConf(existingNetConfig, "network", map[string]interface{}{
		"properties": gardenProps,
	})
}

func ExtractGardenProperties(encodedGardenProperties string) (map[string]string, error) {
	props := make(map[string]string)
	err := json.Unmarshal([]byte(encodedGardenProperties), &props)
	if err != nil {
		return nil, fmt.Errorf("unmarshal garden properties: %s", err)
	}
	return props, nil
}

func InjectConf(original *libcni.NetworkConfig, key string, newValue interface{}) (*libcni.NetworkConfig, error) {
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

func (c *CNIController) Up(namespacePath, handle, encodedGardenProperties string) (*types.Result, error) {
	var result *types.Result
	for i, networkConfig := range c.NetworkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		enhancedNetConfig, err := InjectGardenProperties(networkConfig, encodedGardenProperties)
		if err != nil {
			return nil, fmt.Errorf("adding garden properties to CNI config: %s", err)
		}

		result, err = c.CNIConfig.AddNetwork(enhancedNetConfig, runtimeConfig)
		if err != nil {
			return nil, fmt.Errorf("add network failed: %s", err)
		}
		c.Logger.Info("up-result", lager.Data{"name": networkConfig.Network.Name, "type": networkConfig.Network.Type, "result": result.String()})
	}
	c.Logger.Info("up-complete", lager.Data{"numConfigs": len(c.NetworkConfigs)})

	return result, nil
}

func (c *CNIController) Down(namespacePath, handle, encodedGardenProperties string) error {
	for i, networkConfig := range c.NetworkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		enhancedNetConfig, err := InjectGardenProperties(networkConfig, encodedGardenProperties)
		if err != nil {
			return fmt.Errorf("adding garden properties to CNI config: %s", err)
		}

		err = c.CNIConfig.DelNetwork(enhancedNetConfig, runtimeConfig)
		if err != nil {
			return fmt.Errorf("del network failed: %s", err)
		}

		c.Logger.Info("down-result", lager.Data{"name": networkConfig.Network.Name, "type": networkConfig.Network.Type})
	}
	c.Logger.Info("down-complete", lager.Data{"numConfigs": len(c.NetworkConfigs)})

	return nil
}
