package rules

import (
	"fmt"
	"strconv"

	"code.cloudfoundry.org/lager"
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
func NewRemoteAllowRule(vni int, destinationIP, protocol string, port int, tag string, sourceAppGUID, destinationAppGUID string) GenericRule {
	return GenericRule{
		Properties: []string{
			"-i", fmt.Sprintf("flannel.%d", vni),
			"-d", destinationIP,
			"-p", protocol,
			"--dport", strconv.Itoa(port),
			"-m", "mark", "--mark", fmt.Sprintf("0x%s", tag),
			"-j", "ACCEPT",
			"-m", "comment", "--comment", fmt.Sprintf("src:%s dst:%s", sourceAppGUID, destinationAppGUID),
		},
	}
}

func NewLocalAllowRule(sourceIP, destinationIP, protocol string, port int, sourceAppGUID, destinationAppGUID string) GenericRule {
	return GenericRule{
		Properties: []string{
			"-i", "cni-flannel0",
			"-s", sourceIP,
			"-d", destinationIP,
			"-p", protocol,
			"--dport", strconv.Itoa(port),
			"-j", "ACCEPT",
			"-m", "comment", "--comment", fmt.Sprintf("src:%s dst:%s", sourceAppGUID, destinationAppGUID),
		},
	}
}

func NewGBPTagRule(sourceIP, tag, appGUID string) GenericRule {
	return GenericRule{
		Properties: []string{
			"-s", sourceIP,
			"-j", "MARK", "--set-xmark", fmt.Sprintf("0x%s", tag),
			"-m", "comment", "--comment", fmt.Sprintf("src:%s", appGUID),
		},
	}
}

func NewDefaultEgressRule(localSubnet, overlayNetwork string) GenericRule {
	return GenericRule{
		Properties: []string{
			"-s", localSubnet,
			"!", "-d", overlayNetwork,
			"-j", "MASQUERADE",
		},
	}
}

func NewLogRule(guardConditions []string, name string) GenericRule {
	properties := append(
		guardConditions,
		"-m", "limit", "--limit", "2/min",
		"-j", "LOG",
		"--log-prefix", name,
	)
	return GenericRule{Properties: properties}
}

func NewAcceptExistingLocalRule() GenericRule {
	return GenericRule{
		Properties: []string{
			"-i", "cni-flannel0",
			"-m", "state", "--state", "ESTABLISHED,RELATED",
			"-j", "ACCEPT",
		},
	}
}

func NewDefaultDenyLocalRule(localSubnet string) GenericRule {
	return GenericRule{
		Properties: []string{
			"-i", "cni-flannel0",
			"-s", localSubnet,
			"-d", localSubnet,
			"-j", "REJECT",
		},
	}
}

func NewAcceptExistingRemoteRule(vni int) GenericRule {
	return GenericRule{
		Properties: []string{
			"-i", fmt.Sprintf("flannel.%d", vni),
			"-m", "state", "--state", "ESTABLISHED,RELATED",
			"-j", "ACCEPT",
		},
	}
}

func NewDefaultDenyRemoteRule(vni int) GenericRule {
	return GenericRule{
		Properties: []string{
			"-i", fmt.Sprintf("flannel.%d", vni),
			"-j", "REJECT",
		},
	}
}
