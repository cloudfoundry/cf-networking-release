package lib

import (
	"encoding/json"
	"fmt"
	"lib/rules"

	"github.com/containernetworking/cni/pkg/types"
)

type WrapperConfig struct {
	Datastore        string                 `json:"datastore"`
	IPTablesLockFile string                 `json:"iptables_lock_file"`
	OverlayNetwork   string                 `json:"overlay_network"`
	Delegate         map[string]interface{} `json:"delegate"`
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

func (c *PluginController) DelegateAdd(netconf map[string]interface{}) (*types.Result, error) {
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

func (c *PluginController) DefaultIPMasq(localSubnetCIDR, overlayNetwork string) error {
	rule := rules.NewDefaultEgressRule(localSubnetCIDR, overlayNetwork)

	exists, err := c.IPTables.Exists("nat", "cni-masq", rule)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	err = c.IPTables.NewChain("nat", "cni-masq")
	if err != nil {
		return err
	}

	err = c.IPTables.BulkAppend("nat", "cni-masq", rule)
	if err != nil {
		return err
	}

	return nil
}
