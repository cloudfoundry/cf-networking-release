package adapter

import "net"

type NetAdapter struct{}

func (a *NetAdapter) InterfaceByIndex(index int) (*net.Interface, error) {
	return net.InterfaceByIndex(index)
}
