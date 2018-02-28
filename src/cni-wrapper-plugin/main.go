package main

import (
	"cni-wrapper-plugin/adapter"
	"cni-wrapper-plugin/discover"
	"cni-wrapper-plugin/legacynet"
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

	result030, err := current.NewResultFromResult(result)
	if err != nil {
		return fmt.Errorf("converting result from delegate plugin: %s", err) // not tested
	}

	containerIP := result030.IPs[0].Address.IP

	// Add container metadata info
	store := &datastore.Store{
		Serializer: &serial.Serial{},
		Locker: &filelock.Locker{
			FileLocker: filelock.NewLocker(n.Datastore + "_lock"),
			Mutex:      new(sync.Mutex),
		},
		DataFilePath:    n.Datastore,
		VersionFilePath: n.Datastore + "_version",
		CacheMutex:      new(sync.RWMutex),
	}

	var cniAddData struct {
		Metadata map[string]interface{}
	}
	if err := json.Unmarshal(args.StdinData, &cniAddData); err != nil {
		panic(err) // not tested, this should be impossible
	}

	if err := store.Add(args.ContainerID, containerIP.String(), cniAddData.Metadata); err != nil {
		storeErr := fmt.Errorf("store add: %s", err)
		fmt.Fprintf(os.Stderr, "%s", storeErr)
		fmt.Fprintf(os.Stderr, "cleaning up from error")
		err = pluginController.DelIPMasq(containerIP.String(), n.VTEPName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "during cleanup: removing IP masq: %s", err)
		}

		return storeErr
	}

	// Initialize dns
	var localDNSServers []string
	for _, entry := range n.DNSServers {
		dnsIP := net.ParseIP(entry)
		if dnsIP == nil {
			return fmt.Errorf(`invalid DNS server "%s", must be valid IP address`, entry)
		} else if dnsIP.IsLinkLocalUnicast() {
			localDNSServers = append(localDNSServers, entry)
		}
	}

	defaultInterface := discover.DefaultInterface{
		NetlinkAdapter: &adapter.NetlinkAdapter{},
		NetAdapter:     &adapter.NetAdapter{},
	}
	defaultIfaceName, err := defaultInterface.Name()
	if err != nil {
		return fmt.Errorf("discover default interface name: %s", err) // not tested
	}

	// Initialize NetOut
	netOutProvider := legacynet.NetOut{
		ChainNamer: &legacynet.ChainNamer{
			MaxLength: 28,
		},
		IPTables:              pluginController.IPTables,
		Converter:             &legacynet.NetOutRuleConverter{Logger: os.Stderr},
		ASGLogging:            n.IPTablesASGLogging,
		C2CLogging:            n.IPTablesC2CLogging,
		DeniedLogsPerSec:      n.IPTablesDeniedLogsPerSec,
		AcceptedUDPLogsPerSec: n.IPTablesAcceptedUDPLogsPerSec,
		IngressTag:            n.IngressTag,
		VTEPName:              n.VTEPName,
	}
	if err := netOutProvider.Initialize(args.ContainerID, containerIP, localDNSServers); err != nil {
		return fmt.Errorf("initialize net out: %s", err)
	}

	// Initialize NetIn
	netinProvider := legacynet.NetIn{
		ChainNamer: &legacynet.ChainNamer{
			MaxLength: 28,
		},
		IPTables:          pluginController.IPTables,
		IngressTag:        n.IngressTag,
		HostInterfaceName: defaultIfaceName,
	}
	err = netinProvider.Initialize(args.ContainerID)

	// Create port mappings
	portMappings := n.RuntimeConfig.PortMappings
	for _, netIn := range portMappings {
		if netIn.HostPort <= 0 {
			return fmt.Errorf("cannot allocate port %d", netIn.HostPort)
		}
		if err := netinProvider.AddRule(args.ContainerID, int(netIn.HostPort), int(netIn.ContainerPort), n.InstanceAddress, containerIP.String()); err != nil {
			return fmt.Errorf("adding netin rule: %s", err)
		}
	}

	// Create egress rules
	netOutRules := n.RuntimeConfig.NetOutRules
	if err := netOutProvider.BulkInsertRules(args.ContainerID, netOutRules); err != nil {
		return fmt.Errorf("bulk insert: %s", err) // not tested
	}

	err = pluginController.AddIPMasq(containerIP.String(), n.VTEPName)
	if err != nil {
		return fmt.Errorf("error setting up default ip masq rule: %s", err)
	}

	result030.DNS.Nameservers = n.DNSServers
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
			FileLocker: filelock.NewLocker(n.Datastore + "_lock"),
			Mutex:      new(sync.Mutex),
		},
		DataFilePath:    n.Datastore,
		VersionFilePath: n.Datastore + "_version",
		CacheMutex:      new(sync.RWMutex),
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

	netInProvider := legacynet.NetIn{
		ChainNamer: &legacynet.ChainNamer{
			MaxLength: 28,
		},
		IPTables:   pluginController.IPTables,
		IngressTag: n.IngressTag,
	}

	if err = netInProvider.Cleanup(args.ContainerID); err != nil {
		fmt.Fprintf(os.Stderr, "net in cleanup: %s", err)
	}

	netOutProvider := legacynet.NetOut{
		ChainNamer: &legacynet.ChainNamer{
			MaxLength: 28,
		},
		IPTables:  pluginController.IPTables,
		Converter: &legacynet.NetOutRuleConverter{Logger: os.Stderr},
		VTEPName:  n.VTEPName,
	}

	if err = netOutProvider.Cleanup(args.ContainerID, container.IP); err != nil {
		fmt.Fprintf(os.Stderr, "net out cleanup: %s", err)
	}

	err = pluginController.DelIPMasq(container.IP, n.VTEPName)
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

	iptLocker := &filelock.Locker{
		FileLocker: filelock.NewLocker(iptablesLockFile),
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
	supportedVersions := []string{"0.3.1"}

	skel.PluginMain(cmdAdd, cmdDel, version.PluginSupports(supportedVersions...))
}
