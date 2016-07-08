package rules

import (
	"fmt"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/rule.go --fake-name Rule . Rule
type Rule interface {
	Enforce(string, IPTables, lager.Logger) error
}

type GenericRule struct {
	Properties []string
}

func (r GenericRule) Enforce(chain string, iptables IPTables, logger lager.Logger) error {
	err := iptables.AppendUnique("filter", chain, r.Properties...)
	if err != nil {
		logger.Error("append-rule", err)
		return fmt.Errorf("appending rule: %s", err)
	}

	logger.Info("enforce-rule", lager.Data{
		"chain":      chain,
		"properties": r.Properties,
	})

	return nil
}
