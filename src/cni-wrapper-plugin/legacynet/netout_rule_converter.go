package legacynet

import (
	"fmt"
	"io"
	"lib/rules"

	"code.cloudfoundry.org/garden"
)

type NetOutRuleConverter struct {
	Logger io.Writer
}

func (c *NetOutRuleConverter) BulkConvert(netOutRules []garden.NetOutRule, containerIP, logChainName string, globalLogging bool) []rules.IPTablesRule {
	ruleSpec := []rules.IPTablesRule{}
	for _, rule := range netOutRules {
		for _, t := range c.Convert(rule, containerIP, logChainName, globalLogging) {
			ruleSpec = append(ruleSpec, t)
		}
	}
	return ruleSpec
}

func (c *NetOutRuleConverter) Convert(rule garden.NetOutRule, containerIP, logChainName string, globalLogging bool) []rules.IPTablesRule {
	ruleSpec := []rules.IPTablesRule{}
	for _, network := range rule.Networks {
		startIP, endIP := network.Start.String(), network.End.String()
		protocol := lookupProtocol(rule.Protocol)
		log := rule.Log || globalLogging
		switch rule.Protocol {
		case garden.ProtocolTCP:
			fallthrough
		case garden.ProtocolUDP:
			if len(rule.Ports) == 0 {
				fmt.Fprintf(c.Logger, "UDP/TCP rule must specify ports: %+v\n", rule)
				continue
			}
			for _, portRange := range rule.Ports {
				startPort := int(portRange.Start)
				endPort := int(portRange.End)
				if log {
					ruleSpec = append(ruleSpec, rules.NewNetOutWithPortsLogRule(containerIP, startIP, endIP, startPort, endPort, protocol, logChainName))
				} else {
					ruleSpec = append(ruleSpec, rules.NewNetOutWithPortsRule(containerIP, startIP, endIP, startPort, endPort, protocol))
				}
			}
		case garden.ProtocolICMP:
			if rule.ICMPs == nil || rule.ICMPs.Code == nil {
				fmt.Fprintf(c.Logger, "ICMP rule must specify ICMP type/code: %+v\n", rule)
				continue
			}
			if len(rule.Ports) > 0 {
				fmt.Fprintf(c.Logger, "ICMP rule must not specify ports: %+v\n", rule)
				continue
			}
			icmpType := int(uint8(rule.ICMPs.Type))
			code := rule.ICMPs.Code
			icmpCode := int(uint8(*code))
			if log {
				ruleSpec = append(ruleSpec, rules.NewNetOutICMPLogRule(containerIP, startIP, endIP, icmpType, icmpCode, logChainName))
			} else {
				ruleSpec = append(ruleSpec, rules.NewNetOutICMPRule(containerIP, startIP, endIP, icmpType, icmpCode))
			}
		case garden.ProtocolAll:
			if len(rule.Ports) > 0 {
				fmt.Fprintf(c.Logger, "Rule for all protocols (TCP/UDP/ICMP) must not specify ports: %+v\n", rule)
				continue
			}
			if log {
				ruleSpec = append(ruleSpec, rules.NewNetOutLogRule(containerIP, startIP, endIP, logChainName))
			} else {
				ruleSpec = append(ruleSpec, rules.NewNetOutRule(containerIP, startIP, endIP))
			}
		}
	}
	return ruleSpec
}

func udpOrTcp(protocol garden.Protocol) bool {
	return protocol == garden.ProtocolTCP || protocol == garden.ProtocolUDP
}

func lookupProtocol(protocol garden.Protocol) string {
	switch protocol {
	case garden.ProtocolTCP:
		return "tcp"
	case garden.ProtocolUDP:
		return "udp"
	case garden.ProtocolICMP:
		return "icmp"
	default:
		return "all"
	}
}
