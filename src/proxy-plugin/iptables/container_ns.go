package iptables

import (
	"fmt"
	"lib/rules"
	"proxy-plugin/lib"
)

type ContainerNS struct {
	CommandRunner      lib.CommandRunner
	ContainerNameSpace string
}

func (c ContainerNS) NewChain(table, chain string) error {
	args := append(c.baseArgs(), "-t", table, "-N", chain)
	output, err := c.CommandRunner.Exec("ip", args...)
	if err != nil {
		return fmt.Errorf("new chain: failed running 'ip' with args: %v output: %q err: %q", args, output, err)
	}
	return nil
}

func (c ContainerNS) BulkAppend(table, chain string, rulespecs ...rules.IPTablesRule) error {
	for _, rulespec := range rulespecs {
		args := append(c.baseArgs(), "-t", table, "-A")
		args = append(args, rulespec...)
		output, err := c.CommandRunner.Exec("ip", args...)
		if err != nil {
			return fmt.Errorf("bulk append: failed running 'ip' with args: %v output: %q err: %q", args, output, err)
		}
	}
	return nil
}

func (c ContainerNS) DeleteChain(table, chain string) error {
	args := append(c.baseArgs(), "-t", table, "-X", chain)
	output, err := c.CommandRunner.Exec("ip", args...)
	if err != nil {
		return fmt.Errorf("delete chain: failed running 'ip' with args: %v output: %q err: %q", args, output, err)
	}
	return nil
}

func (c ContainerNS) Delete(table, chain string, rulespec rules.IPTablesRule) error {
	args := append(c.baseArgs(), "-t", table, "-D")
	args = append(args, rulespec...)
	output, err := c.CommandRunner.Exec("ip", args...)
	if err != nil {
		return fmt.Errorf("delete: failed running 'ip' with args: %v output: %q err: %q", args, output, err)
	}
	return nil
}

func (c ContainerNS) Exists(table, chain string, rulespec rules.IPTablesRule) (bool, error) {
	panic("not implemented")
}

func (c ContainerNS) List(table, chain string) ([]string, error) {
	panic("not implemented")
}

func (c ContainerNS) ClearChain(table, chain string) error {
	panic("not implemented")
}

func (c ContainerNS) BulkInsert(table, chain string, pos int, rulespec ...rules.IPTablesRule) error {
	panic("not implemented")
}

func (c ContainerNS) baseArgs() []string {
	return []string{"netns", "exec", c.ContainerNameSpace, "iptables"}
}