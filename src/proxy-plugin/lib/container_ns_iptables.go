package lib

import (
	"fmt"
	"lib/rules"
)

type ContainerNSIPTables struct {
	CommandRunner      CommandRunner
	ContainerNameSpace string
}

func (c ContainerNSIPTables) NewChain(table, chain string) error {
	args := append(c.baseArgs(), "-t", table, "-N", chain)
	output, err := c.CommandRunner.Exec("ip", args...)
	if err != nil {
		return fmt.Errorf("new chain: failed running 'ip' with args: %v output: %q err: %q", args, output, err)
	}
	return nil
}

func (c ContainerNSIPTables) BulkAppend(table, chain string, rulespecs ...rules.IPTablesRule) error {
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

func (c ContainerNSIPTables) DeleteChain(table, chain string) error {
	args := append(c.baseArgs(), "-t", table, "-X", chain)
	output, err := c.CommandRunner.Exec("ip", args...)
	if err != nil {
		return fmt.Errorf("delete chain: failed running 'ip' with args: %v output: %q err: %q", args, output, err)
	}
	return nil
}

func (c ContainerNSIPTables) Delete(table, chain string, rulespec rules.IPTablesRule) error {
	args := append(c.baseArgs(), "-t", table, "-D")
	args = append(args, rulespec...)
	output, err := c.CommandRunner.Exec("ip", args...)
	if err != nil {
		return fmt.Errorf("delete: failed running 'ip' with args: %v output: %q err: %q", args, output, err)
	}
	return nil
}

func (c ContainerNSIPTables) Exists(table, chain string, rulespec rules.IPTablesRule) (bool, error) {
	panic("not implemented")
}

func (c ContainerNSIPTables) List(table, chain string) ([]string, error) {
	panic("not implemented")
}

func (c ContainerNSIPTables) ClearChain(table, chain string) error {
	panic("not implemented")
}

func (c ContainerNSIPTables) BulkInsert(table, chain string, pos int, rulespec ...rules.IPTablesRule) error {
	panic("not implemented")
}

func (c ContainerNSIPTables) baseArgs() []string {
	return []string{"netns", "exec", c.ContainerNameSpace, "iptables"}
}