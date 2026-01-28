package cni

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"io"

	"github.com/containernetworking/cni/libcni"
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

	var confListFilePaths []string

	err := filepath.Walk(l.ConfigDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".conflist") {
			confListFilePaths = append(confListFilePaths, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking config directory: %s", err)
	}

	var toReturn *libcni.NetworkConfigList

	if len(confListFilePaths) > 0 {
		path := confListFilePaths[0]
		confList, err := libcni.ConfListFromFile(path)
		if err != nil {
			return confList, fmt.Errorf("unable to load config from %s: %s", path, err)
		}

		toReturn = confList
	}

	if len(confListFilePaths) > 1 {
		fmt.Fprintf(l.Logger, `%s - Only one CNI conflist (chain) will be executed.
							If multiple CNI config files are present, behavior is undefined.`, time.Now().Format(time.RFC3339))
	}

	return toReturn, nil
}
