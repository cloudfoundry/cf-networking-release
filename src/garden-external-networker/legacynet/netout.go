package legacynet

import (
	"fmt"
	"lib/rules"
	"net"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
)

const prefixNetOut = "netout"
const suffixNetOutLog = "log"

//go:generate counterfeiter -o ../fakes/net_out_rule_converter.go --fake-name NetOutRuleConverter . netOutRuleConverter
type netOutRuleConverter interface {
	Convert(rule garden.NetOutRule, containerIP, logChainName string) []rules.GenericRule
	BulkConvert(rules []garden.NetOutRule, containerIP, logChainName string) []rules.GenericRule
}

type NetOut struct {
	ChainNamer chainNamer
	IPTables   rules.IPTablesExtended
	Converter  netOutRuleConverter
}

func (m *NetOut) Initialize(logger lager.Logger, containerHandle string, containerIP net.IP, overlayNetwork string) error {
	chain := m.ChainNamer.Prefix(prefixNetOut, containerHandle)
	logChain, err := m.ChainNamer.Postfix(chain, suffixNetOutLog)
	if err != nil {
		return fmt.Errorf("getting chain name: %s", err)
	}
	chainsToCreate := []string{chain, logChain}

	for _, c := range chainsToCreate {
		err = m.IPTables.NewChain("filter", c)
		if err != nil {
			return fmt.Errorf("creating chain: %s", err)
		}

		err = m.IPTables.Insert("filter", "FORWARD", 1, []string{"--jump", c}...)
		if err != nil {
			return fmt.Errorf("inserting rule: %s", err)
		}
	}

	defaultRuleSpec := []rules.Rule{
		rules.NewNetOutRelatedEstablishedRule(containerIP.String(), overlayNetwork),
		rules.NewNetOutDefaultRejectRule(containerIP.String(), overlayNetwork),
	}
	logRuleSpec := []rules.Rule{
		rules.NewNetOutDefaultLogRule(containerHandle),
		rules.NewReturnRule(),
	}

	var ruleSpecs []rules.Rule
	for _, c := range chainsToCreate {
		switch c {
		case chain:
			ruleSpecs = defaultRuleSpec
		case logChain:
			ruleSpecs = logRuleSpec
		}
		for _, spec := range ruleSpecs {
			err = spec.Enforce("filter", c, m.IPTables, logger)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *NetOut) Cleanup(containerHandle string) error {
	chain := m.ChainNamer.Prefix(prefixNetOut, containerHandle)
	logChain, err := m.ChainNamer.Postfix(chain, suffixNetOutLog)
	if err != nil {
		return fmt.Errorf("getting chain name: %s", err)
	}

	chainsToClean := []string{chain, logChain}
	for _, c := range chainsToClean {
		err = m.IPTables.Delete("filter", "FORWARD", []string{"--jump", c}...)
		if err != nil {
			return fmt.Errorf("delete rule: %s", err)
		}

		err = m.IPTables.ClearChain("filter", c)
		if err != nil {
			return fmt.Errorf("clear chain: %s", err)
		}

		err = m.IPTables.DeleteChain("filter", c)
		if err != nil {
			return fmt.Errorf("delete chain: %s", err)
		}
	}

	return nil
}

func (m *NetOut) InsertRule(containerHandle string, rule garden.NetOutRule, containerIP string) error {
	chain := m.ChainNamer.Prefix(prefixNetOut, containerHandle)
	logChain, err := m.ChainNamer.Postfix(chain, suffixNetOutLog)
	if err != nil {
		return fmt.Errorf("getting chain name: %s", err)
	}

	ruleSpec := m.Converter.Convert(rule, containerIP, logChain)
	for _, iptRule := range ruleSpec {
		err := m.IPTables.Insert("filter", chain, 1, iptRule.Properties...)
		if err != nil {
			return fmt.Errorf("inserting net-out rule: %s", err)
		}
	}

	return nil
}

func (m *NetOut) BulkInsertRules(containerHandle string, netOutRules []garden.NetOutRule, containerIP string) error {
	chain := m.ChainNamer.Prefix(prefixNetOut, containerHandle)
	logChain, err := m.ChainNamer.Postfix(chain, suffixNetOutLog)
	if err != nil {
		return fmt.Errorf("getting chain name: %s", err)
	}

	ruleSpec := m.Converter.BulkConvert(netOutRules, containerIP, logChain)
	err = m.IPTables.BulkInsert("filter", chain, 1, ruleSpec...)
	if err != nil {
		return fmt.Errorf("bulk inserting net-out rules: %s", err)
	}

	return nil
}
