package lib

import (
	"lib/rules"
	"strconv"
	"fmt"
)

type ProxyRules struct {
	IPTables rules.IPTablesAdapter
	ProxyPort int
	OverlayNetwork string
}

func (p *ProxyRules) Add(proxyChainName string) error {
	err := p.IPTables.NewChain("nat", proxyChainName)
	if err != nil {
		return fmt.Errorf("creating new chain: %s", err)
	}

	chainRules := p.chainRules(proxyChainName)
	err = p.IPTables.BulkAppend("nat", proxyChainName, chainRules...)
	if err != nil {
		return fmt.Errorf("appending rules: %s", err)
	}

	return nil
}

func (p *ProxyRules) Del(proxyChainName string) error {
	chainRules := p.chainRules(proxyChainName)
	for _, rule := range chainRules {
		err := p.IPTables.Delete("nat", proxyChainName, rule)
		if err != nil {
			return fmt.Errorf("deleting rule: %s", err)
		}
	}

	err := p.IPTables.DeleteChain("nat", proxyChainName)
	if err != nil {
		return fmt.Errorf("deleting chain: %s", err)
	}

	return nil
}

func (p *ProxyRules) chainRules(proxyChainName string) []rules.IPTablesRule {
	return []rules.IPTablesRule{
		{"OUTPUT", "-j", proxyChainName},
		{proxyChainName, "-m", "owner", "!", "--uid-owner", "1000", "-j", "RETURN"},
		{proxyChainName, "-d", p.OverlayNetwork, "-p", "tcp", "-j", "REDIRECT", "--to-ports", string(strconv.Itoa(p.ProxyPort))},
	}
}