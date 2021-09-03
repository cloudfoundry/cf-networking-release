package vip

import (
	"crypto/sha256"
	"net"
)

type Provider struct {
	CIDR *net.IPNet
}

func (p *Provider) Get(hostname string) string {
	hasher := sha256.New()
	hasher.Write([]byte(hostname))
	hash := hasher.Sum(nil)

	ip := p.CIDR.IP
	wildcard := net.IPv4Mask(^p.CIDR.Mask[0], ^p.CIDR.Mask[1], ^p.CIDR.Mask[2], ^p.CIDR.Mask[3])
	vip := net.IPv4(
		hash[0]&wildcard[0]|ip[0],
		hash[1]&wildcard[1]|ip[1],
		hash[2]&wildcard[2]|ip[2],
		hash[3]&wildcard[3]|ip[3],
	)

	return vip.String()
}
