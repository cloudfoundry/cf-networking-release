package adapter

import "github.com/containernetworking/plugins/pkg/ns"

type NamespaceAdapter struct{}

func (n *NamespaceAdapter) GetNS(path string) (ns.NetNS, error) {
	return ns.GetNS(path)
}

func (n *NamespaceAdapter) GetCurrentNS() (ns.NetNS, error) {
	return ns.GetCurrentNS()
}
