package legacynet

import (
	"fmt"
	"lib/rules"
	"net"

	multierror "github.com/hashicorp/go-multierror"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
)

const prefixNetOut = "netout"
const suffixNetOutLog = "log"

//go:generate counterfeiter -o ../fakes/net_out_rule_converter.go --fake-name NetOutRuleConverter . netOutRuleConverter
type netOutRuleConverter interface {
	Convert(rule garden.NetOutRule, containerIP, logChainName string) []rules.IPTablesRule
	BulkConvert(rules []garden.NetOutRule, containerIP, logChainName string) []rules.IPTablesRule
}

type NetOut struct {
	ChainNamer chainNamer
	IPTables   rules.IPTablesAdapter
	Converter  netOutRuleConverter
}

func (m *NetOut) Initialize(logger lager.Logger, containerHandle string, containerIP net.IP, overlayNetwork string) error {
	chain := m.ChainNamer.Prefix(prefixNetOut, containerHandle)
	logChain, err := m.ChainNamer.Postfix(chain, suffixNetOutLog)
	if err != nil {
		return fmt.Errorf("getting chain name: %s", err)
	}

	type rulesAndChain struct {
		Chain string
		Rules []rules.IPTablesRule
	}

	args := []rulesAndChain{
		{
			Chain: chain,
			Rules: []rules.IPTablesRule{
				rules.NewNetOutRelatedEstablishedRule(containerIP.String(), overlayNetwork),
				rules.NewNetOutDefaultRejectRule(containerIP.String(), overlayNetwork),
			},
		},
		{
			Chain: logChain,
			Rules: []rules.IPTablesRule{
				rules.NewNetOutDefaultLogRule(containerHandle),
				rules.NewReturnRule(),
			},
		},
	}

	for _, arg := range args {
		err = m.IPTables.NewChain("filter", arg.Chain)
		if err != nil {
			return fmt.Errorf("creating chain: %s", err)
		}

		err = m.IPTables.BulkInsert("filter", "FORWARD", 1, rules.IPTablesRule{"--jump", arg.Chain})
		if err != nil {
			return fmt.Errorf("inserting rule: %s", err)
		}
		err = m.IPTables.BulkAppend("filter", arg.Chain, arg.Rules...)
		if err != nil {
			return fmt.Errorf("appending rule: %s", err)
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

	var result error
	if err := cleanupChain("filter", "FORWARD", chain, m.IPTables); err != nil {
		result = multierror.Append(result, err)
	}
	if err := cleanupChain("filter", "FORWARD", logChain, m.IPTables); err != nil {
		result = multierror.Append(result, err)
	}

	return result
}

func (m *NetOut) InsertRule(containerHandle string, rule garden.NetOutRule, containerIP string) error {
	chain := m.ChainNamer.Prefix(prefixNetOut, containerHandle)
	logChain, err := m.ChainNamer.Postfix(chain, suffixNetOutLog)
	if err != nil {
		return fmt.Errorf("getting chain name: %s", err)
	}

	ruleSpec := m.Converter.Convert(rule, containerIP, logChain)
	err = m.IPTables.BulkInsert("filter", chain, 1, ruleSpec...)
	if err != nil {
		return fmt.Errorf("inserting net-out rule: %s", err)
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
