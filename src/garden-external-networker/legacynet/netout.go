package legacynet

import (
	"fmt"
	"lib/rules"
	"net"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
)

const prefixNetOut = "netout"

type NetOut struct {
	ChainNamer chainNamer
	IPTables   rules.IPTables
}

func (m *NetOut) Initialize(logger lager.Logger, containerHandle string, containerIP net.IP, overlayNetwork string) error {
	chain := m.ChainNamer.Name(prefixNetOut, containerHandle)

	err := m.IPTables.NewChain("filter", chain)
	if err != nil {
		return fmt.Errorf("creating chain: %s", err)
	}

	err = m.IPTables.Insert("filter", "FORWARD", 1, []string{"--jump", chain}...)
	if err != nil {
		return fmt.Errorf("inserting rule: %s", err)
	}

	ruleSpecs := []rules.Rule{
		rules.NewNetOutRelatedEstablishedRule(containerIP.String(), overlayNetwork),
		rules.NewNetOutDefaultRejectRule(containerIP.String(), overlayNetwork),
	}

	for _, spec := range ruleSpecs {
		err = spec.Enforce("filter", chain, m.IPTables, logger)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *NetOut) Cleanup(containerHandle string) error {
	chain := m.ChainNamer.Name(prefixNetOut, containerHandle)

	err := m.IPTables.Delete("filter", "FORWARD", []string{"--jump", chain}...)
	if err != nil {
		return fmt.Errorf("delete rule: %s", err)
	}

	err = m.IPTables.ClearChain("filter", chain)
	if err != nil {
		return fmt.Errorf("clear chain: %s", err)
	}

	err = m.IPTables.DeleteChain("filter", chain)
	if err != nil {
		return fmt.Errorf("delete chain: %s", err)
	}

	return nil
}

func (m *NetOut) InsertRule(containerHandle string, rule garden.NetOutRule, containerIP string) error {
	chain := m.ChainNamer.Name(prefixNetOut, containerHandle)

	ruleSpec := generateRuleSpec(containerHandle, chain, rule, containerIP)
	for _, iptRule := range ruleSpec {
		err := m.IPTables.Insert("filter", chain, 1, iptRule.Properties...)
		if err != nil {
			return fmt.Errorf("inserting net-out rule: %s", err)
		}
	}

	return nil
}

func generateRuleSpec(containerHandle, chain string, rule garden.NetOutRule, containerIP string) []rules.GenericRule {
	ruleSpec := []rules.GenericRule{}
	for _, network := range rule.Networks {
		if len(rule.Ports) > 0 && udpOrTcp(rule.Protocol) {
			for _, portRange := range rule.Ports {
				ruleSpec = append(ruleSpec, rules.NewNetOutWithPortsRule(
					containerIP,
					network.Start.String(),
					network.End.String(),
					int(portRange.Start),
					int(portRange.End),
					lookupProtocol(rule.Protocol),
				),
				)
			}
		} else {
			ruleSpec = append(ruleSpec, rules.NewNetOutRule(
				containerIP,
				network.Start.String(),
				network.End.String(),
			),
			)
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
	default:
		return "all"
	}
}
