package rules

import (
	"fmt"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/rule.go --fake-name Rule . Rule
type Rule interface {
	Enforce(table, chain string, ipt IPTables, logger lager.Logger) error
}

type GenericRule struct {
	Properties []string
}

func (r GenericRule) Enforce(table, chain string, iptables IPTables, logger lager.Logger) error {
	err := iptables.AppendUnique(table, chain, r.Properties...)
	if err != nil {
		logger.Error("append-rule", err)
		return fmt.Errorf("appending rule: %s", err)
	}

	logger.Info("enforce-rule", lager.Data{
		"table":      table,
		"chain":      chain,
		"properties": r.Properties,
	})

	return nil
}
