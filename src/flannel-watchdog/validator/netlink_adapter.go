package validator

import "github.com/vishvananda/netlink"

type NetlinkAdapter struct{}

func (*NetlinkAdapter) LinkByName(ifname string) (netlink.Link, error) {
	return netlink.LinkByName(ifname)
}

func (*NetlinkAdapter) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}
