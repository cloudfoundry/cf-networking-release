package main

import (
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
)

/* Stdin to wrapper
{
  "name": "cni-wrapper",
  "type": "wrapper",
  "datastore": "wrapper",
	"delegate": {
			"name": "cni-flannel",
			"type": "flannel",
			"delegate": {
				"bridge": "cni-flannel0",
				"isDefaultGateway": true,
				"ipMasq": false
      }
   }
}


{
  "name": "local",
  "type": "bridge",
  "bridge": "cni-bridge",
  "isDefaultGateway": true,
  "ipMasq": true,
  "ipam": {
    "type": "host-local",
    "subnet": "10.254.254.0/24"
  }
}

*/
type WrapperConfig struct {
	types.NetConf
	Datastore string                 `json:"datastore"`
	Delegate  map[string]interface{} `json:"delegate"`
}

func loadWrapperConfig(bytes []byte) (*WrapperConfig, error) {
	n := &WrapperConfig{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, fmt.Errorf("failed to load wrapper config: %v", err)
	}
	return n, nil
}

func delegateAdd(netconf map[string]interface{}) error {
	netconfBytes, err := json.Marshal(netconf)
	if err != nil {
		return fmt.Errorf("error serializing delegate netconf: %v", err)
	}

	// fmt.Println("hoi", netconf["type"], string(netconfBytes))
	result, err := invoke.DelegateAdd(netconf["type"].(string), netconfBytes)
	if err != nil {
		return err
	}

	return result.Print()
}

func cmdAdd(args *skel.CmdArgs) error {
	n, err := loadWrapperConfig(args.StdinData)
	if err != nil {
		return err
	}
	return delegateAdd(n.Delegate)
}

// func flannel(args *skel.CmdArgs, command string) error {
// 	execArgs := &invoke.Args{
// 		Command:     command,
// 		ContainerID: args.ContainerID,
// 		NetNS:       args.Netns,
// 		// PluginArgs:  args.Args,
// 		PluginArgs: [][2]string{[2]string{"foo", "bar"}},
// 		IfName:     args.IfName,
// 		Path:       args.Path,
// 	}
// 	_, err := invoke.ExecPluginWithResult("/var/vcap/packages/flannel/bin", args.StdinData, execArgs)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
func cmdDel(args *skel.CmdArgs) error {
	return nil
}

func main() {
	supportedVersions := []string{"0.1.0", "0.2.0"}
	skel.PluginMain(cmdAdd, cmdDel, version.PluginSupports(supportedVersions...))
}
