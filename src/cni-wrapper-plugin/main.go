package main

import (
	"cni-wrapper-plugin/lib"

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
		panic(err)
	}

	return result.Print()
}

func cmdDel(args *skel.CmdArgs) error {
	return nil
}

func main() {
	supportedVersions := []string{"0.1.0", "0.2.0"}
	skel.PluginMain(cmdAdd, cmdDel, version.PluginSupports(supportedVersions...))
}
