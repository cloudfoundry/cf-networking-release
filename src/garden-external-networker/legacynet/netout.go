package legacynet

import (
	"fmt"
	"lib/rules"
	"net"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
)

const prefixNetOut = "netout"

//go:generate counterfeiter -o ../fakes/net_out_rule_converter.go --fake-name NetOutRuleConverter . netOutRuleConverter
type netOutRuleConverter interface {
	Convert(rule garden.NetOutRule, containerIP string) []rules.GenericRule
}

type NetOut struct {
	ChainNamer chainNamer
	IPTables   rules.IPTables
	Converter  netOutRuleConverter
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

	ruleSpec := m.Converter.Convert(rule, containerIP)
	for _, iptRule := range ruleSpec {
		err := m.IPTables.Insert("filter", chain, 1, iptRule.Properties...)
		if err != nil {
			return fmt.Errorf("inserting net-out rule: %s", err)
		}
	}

	return nil
}
