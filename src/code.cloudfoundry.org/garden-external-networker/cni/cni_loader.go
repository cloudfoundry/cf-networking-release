package cni

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/libcni"
	"io"
	"time"
)

type CNILoader struct {
	PluginDir string
	ConfigDir string
	Logger    io.Writer
}

func (l *CNILoader) GetCNIConfig() *libcni.CNIConfig {
	return &libcni.CNIConfig{Path: []string{l.PluginDir}}
}

func (l *CNILoader) GetNetworkConfig() (*libcni.NetworkConfigList, error) {

	var (
		confFilePaths     []string
		confListFilePaths []string
	)

	err := filepath.Walk(l.ConfigDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".conf") {
			confFilePaths = append(confFilePaths, path)
		} else if strings.HasSuffix(path, ".conflist") {
			confListFilePaths = append(confListFilePaths, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error loading config: %s", err)
	}

	var toReturn *libcni.NetworkConfigList

	if len(confListFilePaths) > 0 {
		path := confListFilePaths[0]
		confList, err := libcni.ConfListFromFile(path)
		if err != nil {
			return confList, fmt.Errorf("unable to load config from %s: %s", path, err)
		}

		toReturn = confList
	} else if len(confFilePaths) > 0 {
		path := confFilePaths[0]
		conf, err := libcni.ConfFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("unable to load config from %s: %s", path, err)
		}

		confList, err := libcni.ConfListFromConf(conf)
		if err != nil {
			// untested, unable to cause failure case.
			return nil, fmt.Errorf("unable to upconvert from conf to conf list %s: %s", path, err)
		}

		toReturn = confList
	}

	if (len(confListFilePaths) + len(confFilePaths)) > 1 {
		fmt.Fprintf(l.Logger, `%s - Only one CNI config file or conflist (chain) will be executed. 
							If a conf and conflist file are both present, then the conflist will be executed. 
							If multiple CNI config files are present, behavior is undefined.`, time.Now().Format(time.RFC3339))
	}

	return toReturn, nil
}
