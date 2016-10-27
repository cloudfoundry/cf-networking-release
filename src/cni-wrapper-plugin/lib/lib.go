package lib

import (
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/types"
)

type WrapperConfig struct {
	Datastore string                 `json:"datastore"`
	Delegate  map[string]interface{} `json:"delegate"`
}

func LoadWrapperConfig(bytes []byte) (*WrapperConfig, error) {
	n := &WrapperConfig{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, fmt.Errorf("loading wrapper config: %v", err)
	}

	if n.Datastore == "" {
		return nil, fmt.Errorf("missing datastore path")
	}

	return n, nil
}

type PluginController struct {
	Delegator Delegator
}

func (c *PluginController) DelegateAdd(netconf map[string]interface{}) (*types.Result, error) {
	netconfBytes, err := json.Marshal(netconf)
	if err != nil {
		return nil, fmt.Errorf("serializing delegate netconf: %v", err)
	}

	delegateType, ok := (netconf["type"]).(string)
	if !ok {
		return nil, fmt.Errorf("delegate config is missing type")
	}

	return c.Delegator.DelegateAdd(delegateType, netconfBytes)
}
