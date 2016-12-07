package main

import (
	"cni-wrapper-plugin/lib"
	"encoding/json"
	"fmt"
	"lib/datastore"
	"lib/filelock"
	"lib/rules"
	"lib/serial"
	"net"
	"os"
	"sync"

	"github.com/containernetworking/cni/pkg/skel"
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
		return fmt.Errorf("delegate call: %v", err)
	}

	_, net, err := net.ParseCIDR(fmt.Sprintf("%s/24", result.IP4.IP.IP.String()))
	if err != nil {
		return err
	}

	err = pluginController.DefaultIPMasq(net.String(), n.OverlayNetwork)
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

	if err := store.Add(args.ContainerID, result.IP4.IP.IP.String(), cniAddData.Metadata); err != nil {
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

	pluginController, err := newPluginController(n.IPTablesLockFile)
	if err != nil {
		return err
	}

	if err := pluginController.DelegateDel(n.Delegate); err != nil {
		fmt.Fprintf(os.Stderr, "delegate delete: %s", err)
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
	supportedVersions := []string{"0.1.0", "0.2.0"}

	skel.PluginMain(cmdAdd, cmdDel, version.PluginSupports(supportedVersions...))
}
