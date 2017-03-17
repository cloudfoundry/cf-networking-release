package validator

import (
	"errors"
	"fmt"

	"code.cloudfoundry.org/lager"
	"github.com/vishvananda/netlink"
)

//go:generate counterfeiter -o fakes/netlink_adapter.go --fake-name NetlinkAdapter . netlinkAdapter
type netlinkAdapter interface {
	LinkByName(string) (netlink.Link, error)
	AddrList(netlink.Link, int) ([]netlink.Addr, error)
}

type Bridge struct {
	Logger         lager.Logger
	BridgeName     string
	NetlinkAdapter netlinkAdapter
	found          bool
}

func (b *Bridge) Validate(ip string) error {
	link, err := b.NetlinkAdapter.LinkByName(b.BridgeName)
	if err != nil {
		b.found = false
		b.Logger.Info("no bridge device found")
		return nil
	}

	addr, err := b.NetlinkAdapter.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("listing addresses: %s", err)
	}
	if len(addr) != 1 {
		return fmt.Errorf(`device '%s' does not have one address`, b.BridgeName)
	}

	deviceIP := addr[0].IPNet.String()
	if !b.found {
		b.found = true
		b.Logger.Info("Found bridge", lager.Data{"name": b.BridgeName})
	}

	if ip != deviceIP {
		return errors.New(`This cell must be restarted (run "bosh restart <job>").  Flannel is out of sync with the local bridge.`)
	}

	return nil
}
