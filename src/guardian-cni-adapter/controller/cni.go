package controller

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

func AppendNetworkSpec(existingNetConfig *libcni.NetworkConfig, gardenNetworkSpec string) (*libcni.NetworkConfig, error) {
	config := make(map[string]interface{})
	err := json.Unmarshal(existingNetConfig.Bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal existing network bytes: %s", err)
	}

	if gardenNetworkSpec != "" {
		networkPayloadMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(gardenNetworkSpec), &networkPayloadMap)
		if err != nil {
			return nil, fmt.Errorf("unmarshal garden network spec: %s", err)
		}

		if len(networkPayloadMap) != 0 {
			config["network"] = map[string]interface{}{
				"properties": networkPayloadMap,
			}
		}
	}

	newBytes, err := json.Marshal(config)
	if err != nil {
		return nil, err //Not tested
	}

	return &libcni.NetworkConfig{
		Network: existingNetConfig.Network,
		Bytes:   newBytes,
	}, nil
}

func (c *CNIController) Up(namespacePath, handle, spec string) (*types.Result, error) {
	var result *types.Result
	for i, networkConfig := range c.NetworkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		enhancedNetConfig, err := AppendNetworkSpec(networkConfig, spec)
		if err != nil {
			return nil, fmt.Errorf("adding garden network spec to CNI config: %s", err)
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

func (c *CNIController) Down(namespacePath, handle, spec string) error {
	for i, networkConfig := range c.NetworkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		enhancedNetConfig, err := AppendNetworkSpec(networkConfig, spec)
		if err != nil {
			return fmt.Errorf("adding garden network spec to CNI config: %s", err)
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
