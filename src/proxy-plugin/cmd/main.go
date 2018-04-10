package main

import (
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	"path/filepath"
	"proxy-plugin/lib"
)

func main() {
	supportedVersions := []string{"0.3.1"}
	skel.PluginMain(cmdAdd, cmdDel, version.PluginSupports(supportedVersions...))
}

func cmdAdd(args *skel.CmdArgs) error {
	config, err := lib.LoadProxyConfig(args.StdinData)
	if err != nil {
		return err
	}

	proxyRules := proxyRules(args.Netns, config.OverlayNetwork, config.ProxyPort)
	proxyChainName := proxyChainName(args.ContainerID)
	return proxyRules.Add(proxyChainName)
}

func cmdDel(args *skel.CmdArgs) error {
	config, err := lib.LoadProxyConfig(args.StdinData)
	if err != nil {
		return err
	}

	proxyRules := proxyRules(args.Netns, config.OverlayNetwork, config.ProxyPort)
	proxyChainName := proxyChainName(args.ContainerID)
	return proxyRules.Del(proxyChainName)
}

func proxyRules(containerNetNS, overlayNetwork string, proxyPort int) lib.ProxyRules {
	ipTables := lib.ContainerNSIPTables{
		CommandRunner:      lib.RealCommandRunner{},
		ContainerNameSpace: filepath.Base(containerNetNS),
	}
	return lib.ProxyRules{
		IPTables:       ipTables,
		OverlayNetwork: overlayNetwork,
		ProxyPort:      proxyPort,
	}
}

func proxyChainName(containerID string) string {
	return ("proxy--" + containerID)[:28]
}
