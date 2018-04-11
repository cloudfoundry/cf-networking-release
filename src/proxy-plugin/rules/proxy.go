package rules

import (
	"lib/rules"
	"strconv"
	"fmt"
)

type Proxy struct {
	IPTables rules.IPTablesAdapter
	ProxyPort int
	OverlayNetwork string
}

func (p *Proxy) Add(chainName string) error {
	name := proxyChainName(chainName)
	err := p.IPTables.NewChain("nat", name)
	if err != nil {
		return fmt.Errorf("creating new chain: %s", err)
	}

	chainRules := p.chainRules(name)
	err = p.IPTables.BulkAppend("nat", name, chainRules...)
	if err != nil {
		return fmt.Errorf("appending rules: %s", err)
	}

	return nil
}

func (p *Proxy) Del(chainName string) error {
	name := proxyChainName(chainName)
	chainRules := p.chainRules(name)
	for _, rule := range chainRules {
		err := p.IPTables.Delete("nat", name, rule)
		if err != nil {
			return fmt.Errorf("deleting rule: %s", err)
		}
	}

	err := p.IPTables.DeleteChain("nat", name)
	if err != nil {
		return fmt.Errorf("deleting chain: %s", err)
	}

	return nil
}

func (p *Proxy) chainRules(proxyChainName string) []rules.IPTablesRule {
	return []rules.IPTablesRule{
		{"OUTPUT", "-j", proxyChainName},
		{proxyChainName, "-m", "owner", "!", "--uid-owner", "1000", "-j", "RETURN"},
		{proxyChainName, "-d", p.OverlayNetwork, "-p", "tcp", "-j", "REDIRECT", "--to-ports", string(strconv.Itoa(p.ProxyPort))},
	}
}

func proxyChainName(containerID string) string {
	return ("proxy--" + containerID)[:28]
}