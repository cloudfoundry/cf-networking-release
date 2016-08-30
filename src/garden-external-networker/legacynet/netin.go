package legacynet

import (
	"fmt"
	"lib/rules"

	"code.cloudfoundry.org/lager"
)

const prefixNetIn = "netin"

type NetIn struct {
	ChainNamer
	IPTables rules.IPTables
	Logger   lager.Logger
}

func (m *NetIn) Initialize(containerHandle string) error {
	chain := m.ChainNamer.Name(prefixNetIn, containerHandle)
	err := m.IPTables.NewChain("nat", chain)
	if err != nil {
		return fmt.Errorf("creating chain: %s", err)
	}

	err = m.IPTables.AppendUnique("nat", "PREROUTING", []string{"--jump", chain}...)
	if err != nil {
		return fmt.Errorf("inserting rule: %s", err)
	}
	return nil
}

func (m *NetIn) Cleanup(containerHandle string) error {
	chain := m.ChainNamer.Name(prefixNetIn, containerHandle)

	err := m.IPTables.Delete("nat", "PREROUTING", []string{"--jump", chain}...)
	if err != nil {
		return fmt.Errorf("delete rule: %s", err)
	}

	err = m.IPTables.ClearChain("nat", chain)
	if err != nil {
		return fmt.Errorf("clear chain: %s", err)
	}

	err = m.IPTables.DeleteChain("nat", chain)
	if err != nil {
		return fmt.Errorf("delete chain: %s", err)
	}
	return nil
}

func (m *NetIn) AddRule(containerHandle string,
	hostPort, containerPort int, hostIP, containerIP string) error {

	chainName := m.ChainNamer.Name(prefixNetIn, containerHandle)
	rule := rules.NewNetInRule(containerIP, containerPort, hostIP, hostPort)
	return rule.Enforce("nat", chainName, m.IPTables, m.Logger)
}
