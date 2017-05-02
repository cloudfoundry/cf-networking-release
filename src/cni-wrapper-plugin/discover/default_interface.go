package discover

import (
	"errors"
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

//go:generate counterfeiter -o fakes/netAdapter.go --fake-name NetAdapter . netAdapter
type netAdapter interface {
	InterfaceByIndex(int) (*net.Interface, error)
}

//go:generate counterfeiter -o fakes/netlinkAdapter.go --fake-name NetlinkAdapter . netlinkAdapter
type netlinkAdapter interface {
	RouteList(netlink.Link, int) ([]netlink.Route, error)
}

type DefaultInterface struct {
	NetlinkAdapter netlinkAdapter
	NetAdapter     netAdapter
}

func (di *DefaultInterface) Name() (string, error) {
	routes, err := di.NetlinkAdapter.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return "", fmt.Errorf("route list: %s", err)
	}

	defaultIndex := -1
	for _, r := range routes {
		if r.Dst == nil {
			if defaultIndex != -1 {
				return "", errors.New("multiple possible default routes")
			}
			defaultIndex = r.LinkIndex
		}
	}
	if defaultIndex == -1 {
		return "", errors.New("no default route")
	}

	iface, err := di.NetAdapter.InterfaceByIndex(defaultIndex)
	if err != nil {
		return "", fmt.Errorf("interface by index: %s", err)
	}

	return iface.Name, nil
}
