package main

import (
	"flag"
	"fmt"
	"garden-external-networker/adapter"
	"garden-external-networker/bindmount"
	"garden-external-networker/cni"
	"garden-external-networker/config"
	"garden-external-networker/ipc"
	"garden-external-networker/manager"
	"garden-external-networker/port_allocator"
	"garden-external-networker/proxy"
	"io"
	"lib/rules"
	"lib/serial"
	"os"
	"sync"

	"github.com/coreos/go-iptables/iptables"

	"code.cloudfoundry.org/filelock"
)

var (
	action    string
	handle    string
	cfg       config.Config
	logPrefix = "cfnetworking"
)

func parseArgs(allArgs []string) error {
	var configFilePath string

	flagSet := flag.NewFlagSet("", flag.ContinueOnError)

	flagSet.StringVar(&action, "action", "", "")
	flagSet.StringVar(&handle, "handle", "", "")
	flagSet.StringVar(&configFilePath, "configFile", "", "")

	err := flagSet.Parse(allArgs[1:])
	if err != nil {
		return err
	}

	if configFilePath == "" {
		return fmt.Errorf("missing required flag 'configFile'")
	}

	cfg, err = config.New(configFilePath)
	if err != nil {
		return err
	}

	if len(flagSet.Args()) > 0 {
		return fmt.Errorf("unexpected extra args: %+v", flagSet.Args())
	}

	if handle == "" {
		return fmt.Errorf("missing required flag 'handle'")
	}

	if action == "" {
		return fmt.Errorf("missing required flag 'action'")
	}

	return nil
}

func main() {
	if err := mainWithError(os.Stderr); err != nil {
		if cfg.LogPrefix != "" {
			logPrefix = cfg.LogPrefix
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", logPrefix, err)
		os.Exit(1)
	}
}

func mainWithError(logger io.Writer) error {
	if len(os.Args) == 1 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		return fmt.Errorf("this is a plugin for Garden-runC.  Don't run it directly.")
	}

	err := parseArgs(os.Args)
	if err != nil {
		return fmt.Errorf("parse args: %s", err)
	}

	cniLoader := &cni.CNILoader{
		PluginDir: cfg.CniPluginDir,
		ConfigDir: cfg.CniConfigDir,
		Logger: logger,
	}

	networkConfigList, err := cniLoader.GetNetworkConfig()
	if err != nil {
		return fmt.Errorf("load cni config: %s", err)
	}

	cniController := &cni.CNIController{
		CNIConfig:          cniLoader.GetCNIConfig(),
		NetworkConfigList: networkConfigList,
	}

	mounter := &bindmount.Mounter{}

	locker := filelock.NewLocker(cfg.StateFilePath)
	tracker := &port_allocator.Tracker{
		StartPort: cfg.StartPort,
		Capacity:  cfg.TotalPorts,
	}
	serializer := &serial.Serial{}
	portAllocator := &port_allocator.PortAllocator{
		Tracker:    tracker,
		Serializer: serializer,
		Locker:     locker,
	}

	ipt, err := iptables.New()
	if err != nil {
		panic(err)
	}

	iptLocker := &filelock.Locker{
		FileLocker: filelock.NewLocker(cfg.IPTablesLockFile),
		Mutex:      &sync.Mutex{},
	}
	restorer := &rules.Restorer{}
	lockedIPTables := &rules.LockedIPTables{
		IPTables: ipt,
		Locker:   iptLocker,
		Restorer: restorer,
	}

	namespaceAdapter := &adapter.NamespaceAdapter{}

	proxyRedirect := &proxy.Redirect{
		IPTables:         lockedIPTables,
		NamespaceAdapter: namespaceAdapter,
		RedirectCIDR:     cfg.ProxyRedirectCIDR,
		ProxyPort:        cfg.ProxyPort,
		ProxyUID:         *cfg.ProxyUID,
	}

	manager := &manager.Manager{
		Logger:        logger,
		CNIController: cniController,
		Mounter:       mounter,
		ProxyRedirect: proxyRedirect,
		BindMountRoot: cfg.BindMountDir,
		PortAllocator: portAllocator,
		SearchDomains: cfg.SearchDomains,
	}

	mux := ipc.Mux{
		Up:   manager.Up,
		Down: manager.Down,
	}

	return mux.Handle(action, handle, os.Stdin, os.Stdout)
}
