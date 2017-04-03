package main

import (
	"flag"
	"fmt"
	"garden-external-networker/bindmount"
	"garden-external-networker/cni"
	"garden-external-networker/config"
	"garden-external-networker/ipc"
	"garden-external-networker/manager"
	"garden-external-networker/port_allocator"
	"io"
	"lib/filelock"
	"lib/serial"
	"os"
)

var (
	action string
	handle string
	cfg    config.Config
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
	if len(flagSet.Args()) > 0 {
		return fmt.Errorf("unexpected extra args: %+v", flagSet.Args())
	}

	if handle == "" {
		return fmt.Errorf("missing required flag 'handle'")
	}

	if configFilePath == "" {
		return fmt.Errorf("missing required flag 'configFile'")
	}

	cfg, err = config.New(configFilePath)
	if err != nil {
		return err
	}

	if action == "" {
		return fmt.Errorf("missing required flag 'action'")
	}

	return nil
}

func main() {
	if err := mainWithError(os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
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
	}

	networks, err := cniLoader.GetNetworkConfigs()
	if err != nil {
		return fmt.Errorf("load cni config: %s", err)
	}

	cniController := &cni.CNIController{
		CNIConfig:      cniLoader.GetCNIConfig(),
		NetworkConfigs: networks,
	}

	mounter := &bindmount.Mounter{}

	locker := &filelock.Locker{Path: cfg.StateFilePath}
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

	manager := &manager.Manager{
		Logger:        logger,
		CNIController: cniController,
		Mounter:       mounter,
		BindMountRoot: cfg.BindMountDir,
		PortAllocator: portAllocator,
	}

	mux := ipc.Mux{
		Up:   manager.Up,
		Down: manager.Down,
	}

	return mux.Handle(action, handle, os.Stdin, os.Stdout)
}
