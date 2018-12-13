package api

import (
	"bytes"
	"errors"
	"fmt"
	"net"
)

type EgressDestinationsValidator struct{}

func (v *EgressDestinationsValidator) ValidateEgressDestinations(destinations []EgressDestination) error {
	if len(destinations) == 0 {
		return errors.New("missing destinations")
	}

	for _, destination := range destinations {

		if destination.Name == "" {
			return errors.New("missing destination name")
		}

		if len(destination.Rules) == 0 {
			return errors.New("missing rules")
		}

		for _, rule := range destination.Rules {
			if rule.Protocol == "" {
				return errors.New("missing destination protocol")
			}

			if !isValidProtocol(rule.Protocol) {
				return fmt.Errorf("invalid destination protocol '%s', specify either tcp, udp, or icmp", rule.Protocol)
			}

			if rule.Protocol != "icmp" && len(rule.Ports) == 0 {
				return errors.New("missing destination ports")
			}

			if rule.Protocol == "icmp" && len(rule.Ports) > 0 {
				return errors.New("ports are not supported for icmp protocol")
			}

			if len(rule.Ports) > 1 {
				return errors.New("only one port range is currently supported")
			}

			for _, portRange := range rule.Ports {
				if portRange.Start > portRange.End {
					return fmt.Errorf("invalid port range %d-%d, start must be less than or equal to end", portRange.Start, portRange.End)
				}

				if portRange.End > 65535 {
					return fmt.Errorf("invalid end port %d, must be in range 1-65535", portRange.End)
				}

				if portRange.Start <= 0 {
					return fmt.Errorf("invalid start port %d, must be in range 1-65535", portRange.Start)
				}
			}

			if rule.Protocol != "icmp" && rule.ICMPCode != nil {
				return fmt.Errorf("invalid destination: cannot set icmp_code property for destination with protocol '%s'", rule.Protocol)
			}

			if rule.Protocol != "icmp" && rule.ICMPType != nil {
				return fmt.Errorf("invalid destination: cannot set icmp_type property for destination with protocol '%s'", rule.Protocol)
			}

			if len(rule.IPRanges) == 0 {
				return errors.New("missing destination IP range")
			}

			if len(rule.IPRanges) > 1 {
				return errors.New("only one IP range is currently supported")
			}

			for _, ipRange := range rule.IPRanges {
				startIP := net.ParseIP(ipRange.Start)
				if startIP == nil || startIP.To4() == nil {
					return fmt.Errorf("invalid ip address '%s', must be a valid IPv4 address", ipRange.Start)
				}

				endIP := net.ParseIP(ipRange.End)
				if endIP == nil || endIP.To4() == nil {
					return fmt.Errorf("invalid ip address '%s', must be a valid IPv4 address", ipRange.End)
				}

				if bytes.Compare(startIP, endIP) > 0 {
					return fmt.Errorf("invalid IP range %s-%s, start must be less than or equal to end", ipRange.Start, ipRange.End)
				}
			}
		}
	}

	return nil
}

func isValidProtocol(protocol string) bool {
	validProtocols := []string{"tcp", "udp", "icmp", "all"}

	for _, validProtocol := range validProtocols {
		if validProtocol == protocol {
			return true
		}
	}
	return false
}
