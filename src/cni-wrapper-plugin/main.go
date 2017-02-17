package main

import (
	"cni-wrapper-plugin/lib"
	"encoding/json"
	"fmt"
	"lib/datastore"
	"lib/filelock"
	"lib/rules"
	"lib/serial"
	"os"
	"sync"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types/020"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/coreos/go-iptables/iptables"
)

func cmdAdd(args *skel.CmdArgs) error {
	n, err := lib.LoadWrapperConfig(args.StdinData)
	if err != nil {
		return err
	}

	pluginController, err := newPluginController(n.IPTablesLockFile)
	if err != nil {
		return err
	}

	result, err := pluginController.DelegateAdd(n.Delegate)
	if err != nil {
		return fmt.Errorf("delegate call: %s", err)
	}

	result020, err := result.GetAsVersion("0.2.0")
	if err != nil {
		return fmt.Errorf("cni delegate plugin result version incompatible: %s", err) // not tested
	}

	containerIP := result020.(*types020.Result).IP4.IP.IP.String()
	err = pluginController.AddIPMasq(containerIP, n.OverlayNetwork)
	if err != nil {
		return fmt.Errorf("error setting up default ip masq rule: %s", err)
	}

	store := &datastore.Store{
		Serializer: &serial.Serial{},
		Locker: &filelock.Locker{
			Path: n.Datastore,
		},
	}

	var cniAddData struct {
		Metadata map[string]interface{}
	}
	if err := json.Unmarshal(args.StdinData, &cniAddData); err != nil {
		panic(err) // not tested, this should be impossible
	}

	if err := store.Add(args.ContainerID, containerIP, cniAddData.Metadata); err != nil {
		storeErr := fmt.Errorf("store add: %s", err)
		fmt.Fprintf(os.Stderr, "%s", storeErr)
		fmt.Fprintf(os.Stderr, "cleaning up from error")
		err = pluginController.DelIPMasq(containerIP, n.OverlayNetwork)
		if err != nil {
			fmt.Fprintf(os.Stderr, "during cleanup: removing IP masq: %s", err)
		}

		return storeErr
	}

	result030, err := current.NewResultFromResult(result020)
	if err != nil {
		return fmt.Errorf("error converting result to 0.3.0: %s", err) // not tested
	}
	return result030.Print()
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

	container, err := store.Delete(args.ContainerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "store delete: %s", err)
	}

	pluginController, err := newPluginController(n.IPTablesLockFile)
	if err != nil {
		return err
	}

	if err := pluginController.DelegateDel(n.Delegate); err != nil {
		fmt.Fprintf(os.Stderr, "delegate delete: %s", err)
	}

	err = pluginController.DelIPMasq(container.IP, n.OverlayNetwork)
	if err != nil {
		fmt.Fprintf(os.Stderr, "removing IP masq: %s", err)
	}

	return nil
}

func newPluginController(iptablesLockFile string) (*lib.PluginController, error) {
	ipt, err := iptables.New()
	if err != nil {
		return nil, err
	}

	iptLocker := &rules.IPTablesLocker{
		FileLocker: &filelock.Locker{Path: iptablesLockFile},
		Mutex:      &sync.Mutex{},
	}
	restorer := &rules.Restorer{}
	lockedIPTables := &rules.LockedIPTables{
		IPTables: ipt,
		Locker:   iptLocker,
		Restorer: restorer,
	}

	pluginController := &lib.PluginController{
		Delegator: lib.NewDelegator(),
		IPTables:  lockedIPTables,
	}
	return pluginController, nil
}

func main() {
	supportedVersions := []string{"0.3.0"}

	skel.PluginMain(cmdAdd, cmdDel, version.PluginSupports(supportedVersions...))
}
