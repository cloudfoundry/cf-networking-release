package cni

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/libcni"
)

type CNILoader struct {
	PluginDir string
	ConfigDir string
}

func (l *CNILoader) GetCNIConfig() *libcni.CNIConfig {
	return &libcni.CNIConfig{Path: []string{l.PluginDir}}
}

func (l *CNILoader) GetNetworkConfigs() ([]*libcni.NetworkConfigList, error) {
	networkListConfigs := []*libcni.NetworkConfigList{}
	err := filepath.Walk(l.ConfigDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".conf") {
			conf, err := libcni.ConfFromFile(path)
			if err != nil {
				return fmt.Errorf("unable to load config from %s: %s", path, err)
			}

			confList, err := libcni.ConfListFromConf(conf)
			if err != nil {
				// untested, unable to cause failure case.
				return fmt.Errorf("unable to upconvert from conf to conf list %s: %s", path, err)
			}

			networkListConfigs = append(networkListConfigs, confList)
			return nil
		}

		if strings.HasSuffix(path, ".conflist") {
			confList, err := libcni.ConfListFromFile(path)
			if err != nil {
				return fmt.Errorf("unable to load config from %s: %s", path, err)
			}

			networkListConfigs = append(networkListConfigs, confList)
			return nil
		}

		return nil
	})

	if err != nil {
		return networkListConfigs, fmt.Errorf("error loading config: %s", err)
	}

	return networkListConfigs, nil
}
