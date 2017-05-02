package adapter

import "github.com/vishvananda/netlink"

type NetlinkAdapter struct{}

func (a *NetlinkAdapter) RouteList(link netlink.Link, family int) ([]netlink.Route, error) {
	return netlink.RouteList(link, family)
}
