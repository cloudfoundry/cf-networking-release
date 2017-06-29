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
	Delegate           map[string]interface{} `json:"delegate"`
	HealthCheckURL     string                 `json:"health_check_url"`
	InstanceAddress    string                 `json:"instance_address"`
	DNSServers         []string               `json:"dns_servers"`
	IPTablesASGLogging bool                   `json:"iptables_asg_logging"`
	IPTablesC2CLogging bool                   `json:"iptables_c2c_logging"`
	IngressTag         string                 `json:"ingress_tag"`
	VTEPName           string                 `json:"vtep_name"`
	RuntimeConfig      RuntimeConfig          `json:"runtimeConfig,omitempty"`
	DeniedLogsPerSec   int                    `json:"denied_logs_per_sec" validate:"min=1"`
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

	if n.HealthCheckURL == "" {
		return nil, fmt.Errorf("missing health check url")
	}

	if n.InstanceAddress == "" {
		return nil, fmt.Errorf("missing instance address")
	}

	if n.IngressTag == "" {
		return nil, fmt.Errorf("missing ingress tag")
	}

	if n.VTEPName == "" {
		return nil, fmt.Errorf("missing vtep device name")
	}

	if n.DeniedLogsPerSec <= 0 {
		return nil, fmt.Errorf("invalid denied logs per sec")
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

func (c *PluginController) AddIPMasq(ip, deviceName string) error {
	rule := rules.NewDefaultEgressRule(ip, deviceName)

	if err := c.IPTables.BulkAppend("nat", "POSTROUTING", rule); err != nil {
		return err
	}

	return nil
}

func (c *PluginController) DelIPMasq(ip, deviceName string) error {
	rule := rules.NewDefaultEgressRule(ip, deviceName)

	if err := c.IPTables.Delete("nat", "POSTROUTING", rule); err != nil {
		return err
	}

	return nil
}
