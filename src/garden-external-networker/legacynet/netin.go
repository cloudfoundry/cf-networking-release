package legacynet

import (
	"fmt"
	"lib/rules"
)

const prefixNetIn = "netin"

type NetIn struct {
	ChainNamer chainNamer
	IPTables   rules.IPTables
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
	err := m.IPTables.AppendUnique("nat", chainName, []string{
		"-d", hostIP, "-p", "tcp",
		"-m", "tcp", "--dport", fmt.Sprintf("%d", hostPort),
		"--jump", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", containerIP, containerPort),
	}...)
	if err != nil {
		return fmt.Errorf("inserting rule: %s", err)
	}

	return nil
}
