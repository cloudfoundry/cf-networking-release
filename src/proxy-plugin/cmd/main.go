package main

import (
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	"path/filepath"
	"proxy-plugin/iptables"
	"proxy-plugin/lib"
	"proxy-plugin/rules"
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

	proxyRules := proxyRules(args.Netns, config.ProxyRange, config.ProxyPort)
	proxyChainName := proxyChainName(args.ContainerID)
	return proxyRules.Add(proxyChainName)
}

func cmdDel(args *skel.CmdArgs) error {
	config, err := lib.LoadProxyConfig(args.StdinData)
	if err != nil {
		return err
	}

	proxyRules := proxyRules(args.Netns, config.ProxyRange, config.ProxyPort)
	proxyChainName := proxyChainName(args.ContainerID)
	return proxyRules.Del(proxyChainName)
}

func proxyRules(containerNetNS, overlayNetwork string, proxyPort int) rules.Proxy {
	ipTables := iptables.ContainerNS{
		CommandRunner:      lib.RealCommandRunner{},
		ContainerNameSpace: filepath.Base(containerNetNS),
	}
	return rules.Proxy{
		IPTables:       ipTables,
		OverlayNetwork: overlayNetwork,
		ProxyPort:      proxyPort,
	}
}

func proxyChainName(containerID string) string {
	return ("proxy--" + containerID)[:28]
}
