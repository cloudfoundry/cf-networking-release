package main

import (
	"cni-wrapper-plugin/lib"
	"cni-wrapper-plugin/lib/datastore"
	"encoding/json"
	"fmt"
	"lib/filelock"
	"lib/serial"
	"os"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
)

func cmdAdd(args *skel.CmdArgs) error {
	n, err := lib.LoadWrapperConfig(args.StdinData)
	if err != nil {
		return err
	}

	pluginController := &lib.PluginController{
		Delegator: lib.NewDelegator(),
	}

	result, err := pluginController.DelegateAdd(n.Delegate)
	if err != nil {
		return fmt.Errorf("delegate call: %v", err)
	}

	store := &datastore.Store{
		Serializer: &serial.Serial{},
		Locker: &filelock.Locker{
			Path: n.Datastore,
		},
	}

	var metadata struct {
		Network struct {
			Properties map[string]interface{}
		}
	}
	if err := json.Unmarshal(args.StdinData, &metadata); err != nil {
		panic(err) // not tested, this should be impossible
	}

	if err := store.Add(args.ContainerID, result.IP4.IP.IP.String(), metadata.Network.Properties); err != nil {
		return fmt.Errorf("store add: %s", err)
	}

	return result.Print()
}

func cmdDel(args *skel.CmdArgs) error {
	n, err := lib.LoadWrapperConfig(args.StdinData)
	if err != nil {
		return err
	}

	store := &datastore.Store{
		Serializer: &serial.Serial{},
		Locker: &filelock.Locker{
			Path: n.Datastore,
		},
	}

	if err := store.Delete(args.ContainerID); err != nil {
		fmt.Fprintf(os.Stderr, "store delete: %s", err)
	}

	pluginController := &lib.PluginController{
		Delegator: lib.NewDelegator(),
	}

	if err := pluginController.DelegateDel(n.Delegate); err != nil {
		fmt.Fprintf(os.Stderr, "delegate delete: %s", err)
	}

	return nil
}

func main() {
	supportedVersions := []string{"0.1.0", "0.2.0"}

	skel.PluginMain(cmdAdd, cmdDel, version.PluginSupports(supportedVersions...))
}
