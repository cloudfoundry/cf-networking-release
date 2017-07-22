package main

import (
	"cni-wrapper-plugin/lib"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/silk/lib/adapter"
)

func main() {
	logger := lager.NewLogger("cni-teardown")
	sink := lager.NewWriterSink(os.Stdout, lager.INFO)
	logger.RegisterSink(sink)

	logger.Info("starting")
	netlinkAdapter := &adapter.NetlinkAdapter{}

	links, err := netlinkAdapter.LinkList()
	if err != nil {
		logger.Error("failed-to-list-network-devices", err) // not tested
	}

	for _, link := range links {
		if link.Type() == "ifb" && strings.HasPrefix(link.Attrs().Name, "i") {
			err = netlinkAdapter.LinkDel(link)
			if err != nil {
				logger.Error("failed-to-remove-ifb", err)
			}
		}
	}

	configFilePath := flag.String("config", "", "path to config file")
	flag.Parse()

	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		logger.Error("load-config-file", err)
		os.Exit(1)
	}

	cfg, err := lib.LoadWrapperConfig(configBytes)
	if err != nil {
		logger.Error("read-config-file", err)
		os.Exit(1)
	}

	containerMetadataDir := filepath.Dir(cfg.Datastore)
	err = os.RemoveAll(containerMetadataDir)
	if err != nil {
		logger.Info("failed-to-remove-datastore-path", lager.Data{"path": containerMetadataDir, "err": err})
	}

	if delegateDataDirPath, ok := cfg.Delegate["dataDir"].(string); ok {
		err = os.RemoveAll(delegateDataDirPath)
		if err != nil {
			logger.Info("failed-to-remove-delegate-datastore-path", lager.Data{"path": delegateDataDirPath, "err": err})
		}
	}

	if delegateDataStorePath, ok := cfg.Delegate["datastore"].(string); ok {
		silkDir := filepath.Dir(delegateDataStorePath)
		err = os.RemoveAll(silkDir)
		if err != nil {
			logger.Info("failed-to-remove-delegate-data-dir-path", lager.Data{"path": silkDir, "err": err})
		}
	}

	logger.Info("complete")
}
