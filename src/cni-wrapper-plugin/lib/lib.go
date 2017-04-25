package lib

import (
	"encoding/json"
	"fmt"
	"lib/rules"

	"code.cloudfoundry.org/garden"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
)

type RuntimeConfig struct {
	PortMappings []garden.NetIn      `json:"portMappings"`
	NetOutRules  []garden.NetOutRule `json:"netOutRules"`
}

type WrapperConfig struct {
	Datastore          string                 `json:"datastore"`
	IPTablesLockFile   string                 `json:"iptables_lock_file"`
	OverlayNetwork     string                 `json:"overlay_network"`
	Delegate           map[string]interface{} `json:"delegate"`
	HealthCheckURL     string                 `json:"health_check_url"`
	InstanceAddress    string                 `json:"instance_address"`
	DNSServers         []string               `json:"dns_servers"`
	IPTablesASGLogging bool                   `json:"iptables_asg_logging"`
	IPTablesC2CLogging bool                   `json:"iptables_c2c_logging"`
	RuntimeConfig      RuntimeConfig          `json:"runtimeConfig,omitempty"`
}

func LoadWrapperConfig(bytes []byte) (*WrapperConfig, error) {
	n := &WrapperConfig{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, fmt.Errorf("loading wrapper config: %v", err)
	}

	if n.Datastore == "" {
		return nil, fmt.Errorf("missing datastore path")
	}

	if n.IPTablesLockFile == "" {
		return nil, fmt.Errorf("missing iptables lock file path")
	}

	if n.OverlayNetwork == "" {
		return nil, fmt.Errorf("missing overlay network")
	}

	if n.HealthCheckURL == "" {
		return nil, fmt.Errorf("missing health check url")
	}

	if n.InstanceAddress == "" {
		return nil, fmt.Errorf("missing instance address")
	}

	if _, ok := n.Delegate["cniVersion"]; !ok {
		n.Delegate["cniVersion"] = version.Current()
	}

	return n, nil
}

type PluginController struct {
	Delegator Delegator
	IPTables  rules.IPTablesAdapter
}

func getDelegateParams(netconf map[string]interface{}) (string, []byte, error) {
	netconfBytes, err := json.Marshal(netconf)
	if err != nil {
		return "", nil, fmt.Errorf("serializing delegate netconf: %v", err)
	}

	delegateType, ok := (netconf["type"]).(string)
	if !ok {
		return "", nil, fmt.Errorf("delegate config is missing type")
	}

	return delegateType, netconfBytes, nil
}

func (c *PluginController) DelegateAdd(netconf map[string]interface{}) (types.Result, error) {
	delegateType, netconfBytes, err := getDelegateParams(netconf)
	if err != nil {
		return nil, err
	}

	return c.Delegator.DelegateAdd(delegateType, netconfBytes)
}

func (c *PluginController) DelegateDel(netconf map[string]interface{}) error {
	delegateType, netconfBytes, err := getDelegateParams(netconf)
	if err != nil {
		return err
	}

	return c.Delegator.DelegateDel(delegateType, netconfBytes)
}

func (c *PluginController) AddIPMasq(ip, overlayNetwork string) error {
	rule := rules.NewDefaultEgressRule(ip, overlayNetwork)

	if err := c.IPTables.BulkAppend("nat", "POSTROUTING", rule); err != nil {
		return err
	}

	return nil
}

func (c *PluginController) DelIPMasq(ip, overlayNetwork string) error {
	rule := rules.NewDefaultEgressRule(ip, overlayNetwork)

	if err := c.IPTables.Delete("nat", "POSTROUTING", rule); err != nil {
		return err
	}

	return nil
}
