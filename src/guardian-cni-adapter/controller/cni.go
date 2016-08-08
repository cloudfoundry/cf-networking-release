package controller

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/lager"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
)

type CNIController struct {
	PluginDir string
	ConfigDir string
	Logger    lager.Logger

	cniConfig      *libcni.CNIConfig
	networkConfigs []*libcni.NetworkConfig
}

func (c *CNIController) ensureInitialized() error {
	if c.cniConfig == nil {
		c.cniConfig = &libcni.CNIConfig{Path: []string{c.PluginDir}}
	}

	if c.networkConfigs == nil {
		c.networkConfigs = []*libcni.NetworkConfig{}

		err := filepath.Walk(c.ConfigDir, func(path string, info os.FileInfo, err error) error {
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
			c.networkConfigs = append(c.networkConfigs, conf)
			c.Logger.Info("loaded-config", lager.Data{"network": conf.Network, "raw": string(conf.Bytes)})
			return nil
		})
		if err != nil {
			return fmt.Errorf("error loading config: %s", err)
		}
	}

	return nil
}

func isCIDR(spec string) bool {
	_, _, err := net.ParseCIDR(spec)
	return err == nil
}

func isIP(spec string) bool {
	ip := net.ParseIP(spec)
	return ip != nil
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
	err := c.ensureInitialized()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize controller: %s", err)
	}

	var result *types.Result
	for i, networkConfig := range c.networkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		enhancedNetConfig, err := AppendNetworkSpec(networkConfig, spec)
		if err != nil {
			return nil, fmt.Errorf("adding garden network spec to CNI config: %s", err)
		}

		result, err = c.cniConfig.AddNetwork(enhancedNetConfig, runtimeConfig)
		if err != nil {
			return nil, fmt.Errorf("add network failed: %s", err)
		}
		c.Logger.Info("up-result", lager.Data{"name": networkConfig.Network.Name, "type": networkConfig.Network.Type, "result": result.String()})
	}
	c.Logger.Info("up-complete", lager.Data{"numConfigs": len(c.networkConfigs)})

	return result, nil
}

func (c *CNIController) Down(namespacePath, handle, spec string) error {
	err := c.ensureInitialized()
	if err != nil {
		return fmt.Errorf("failed to initialize controller: %s", err)
	}

	for i, networkConfig := range c.networkConfigs {
		runtimeConfig := &libcni.RuntimeConf{
			ContainerID: handle,
			NetNS:       namespacePath,
			IfName:      fmt.Sprintf("eth%d", i),
		}

		enhancedNetConfig, err := AppendNetworkSpec(networkConfig, spec)
		if err != nil {
			return fmt.Errorf("adding garden network spec to CNI config: %s", err)
		}

		err = c.cniConfig.DelNetwork(enhancedNetConfig, runtimeConfig)
		if err != nil {
			return fmt.Errorf("del network failed: %s", err)
		}

		c.Logger.Info("down-result", lager.Data{"name": networkConfig.Network.Name, "type": networkConfig.Network.Type})
	}
	c.Logger.Info("down-complete", lager.Data{"numConfigs": len(c.networkConfigs)})

	return nil
}
