package proxy

import (
	"fmt"
	"lib/rules"

	"github.com/containernetworking/plugins/pkg/ns"
)

//go:generate counterfeiter -o ../fakes/namespaceAdapter.go --fake-name NamespaceAdapter . namespaceAdapter
type namespaceAdapter interface {
	GetNS(netNamespace string) (ns.NetNS, error)
}

type Redirect struct {
	IPTables                   rules.IPTablesAdapter
	NamespaceAdapter           namespaceAdapter
	RedirectCIDR               string
	ProxyPort                  int
	ProxyUID                   int
	EnableIngressProxyRedirect bool
}

func (r *Redirect) Apply(containerNetNamespace string) error {
	netNS, err := r.NamespaceAdapter.GetNS(containerNetNamespace)
	err = netNS.Do(func(_ ns.NetNS) error {
		if r.RedirectCIDR != "" {
			err := r.IPTables.BulkAppend("nat", "OUTPUT", rules.IPTablesRule{
				"-d", r.RedirectCIDR,
				"-p", "tcp",
				"-j", "REDIRECT", "--to-port", fmt.Sprintf("%d", r.ProxyPort),
			})
			if err != nil {
				return err
			}
		}

		if r.EnableIngressProxyRedirect {
			err := r.IPTables.BulkAppend("nat", "PREROUTING", rules.IPTablesRule{
				"-p", "tcp",
				"-j", "REDIRECT", "--to-port", fmt.Sprintf("%d", r.ProxyPort),
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("do in container: %s", err)
	}

	return nil
}
