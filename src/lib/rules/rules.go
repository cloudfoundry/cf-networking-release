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

//go:generate counterfeiter -o ../fakes/iptables.go --fake-name IPTables . IPTables
type IPTables interface {
	Exists(table, chain string, rulespec ...string) (bool, error)
	Insert(table, chain string, pos int, rulespec ...string) error
	AppendUnique(table, chain string, rulespec ...string) error
	Delete(table, chain string, rulespec ...string) error
	List(table, chain string) ([]string, error)
	NewChain(table, chain string) error
	ClearChain(table, chain string) error
	DeleteChain(table, chain string) error
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

	logger.Debug("enforce-rule", lager.Data{
		"table":      table,
		"chain":      chain,
		"properties": fmt.Sprintf("%s", r.Properties),
	})

	return nil
}

func NewMarkAllowRule(destinationIP, protocol string, port int, tag string, sourceAppGUID, destinationAppGUID string) GenericRule {
	return GenericRule{
		Properties: []string{
			"-d", destinationIP,
			"-p", protocol,
			"--dport", strconv.Itoa(port),
			"-m", "mark", "--mark", fmt.Sprintf("0x%s", tag),
			"--jump", "ACCEPT",
			"-m", "comment", "--comment", fmt.Sprintf("src:%s dst:%s", sourceAppGUID, destinationAppGUID),
		},
	}
}

func NewMarkSetRule(sourceIP, tag, appGUID string) GenericRule {
	return GenericRule{
		Properties: []string{
			"--source", sourceIP,
			"--jump", "MARK", "--set-xmark", fmt.Sprintf("0x%s", tag),
			"-m", "comment", "--comment", fmt.Sprintf("src:%s", appGUID),
		},
	}
}

func NewDefaultEgressRule(localSubnet, overlayNetwork string) GenericRule {
	return GenericRule{
		Properties: []string{
			"--source", localSubnet,
			"!", "-d", overlayNetwork,
			"--jump", "MASQUERADE",
		},
	}
}

func NewLogRule(guardConditions []string, name string) GenericRule {
	properties := append(
		guardConditions,
		"-m", "limit", "--limit", "2/min",
		"--jump", "LOG",
		"--log-prefix", name,
	)
	return GenericRule{Properties: properties}
}

func NewAcceptExistingLocalRule() GenericRule {
	return GenericRule{
		Properties: []string{
			"-i", "cni-flannel0",
			"-m", "state", "--state", "ESTABLISHED,RELATED",
			"--jump", "ACCEPT",
		},
	}
}

func NewDefaultDenyLocalRule(localSubnet string) GenericRule {
	return GenericRule{
		Properties: []string{
			"-i", "cni-flannel0",
			"--source", localSubnet,
			"-d", localSubnet,
			"--jump", "REJECT",
		},
	}
}

func NewAcceptExistingRemoteRule(vni int) GenericRule {
	return GenericRule{
		Properties: []string{
			"-i", fmt.Sprintf("flannel.%d", vni),
			"-m", "state", "--state", "ESTABLISHED,RELATED",
			"--jump", "ACCEPT",
		},
	}
}

func NewDefaultDenyRemoteRule(vni int) GenericRule {
	return GenericRule{
		Properties: []string{
			"-i", fmt.Sprintf("flannel.%d", vni),
			"--jump", "REJECT",
		},
	}
}

func NewNetOutRule(containerIP string, startIP string, endIP string) GenericRule {
	return GenericRule{
		Properties: []string{
			"--source", containerIP,
			"-m", "iprange",
			"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
			"--jump", "RETURN",
			// "-m", "comment", "--comment", fmt.Sprintf("dst:%s", groupID),
		},
	}
}

func NewNetOutWithPortsRule(containerIP string, startIP string, endIP string, startPort int, endPort int, protocol string) GenericRule {
	return GenericRule{
		Properties: []string{
			"--source", containerIP,
			"-m", "iprange",
			"-p", protocol,
			"--dst-range", fmt.Sprintf("%s-%s", startIP, endIP),
			"-m", protocol,
			"--destination-port", fmt.Sprintf("%d:%d", startPort, endPort),
			"--jump", "RETURN",
		},
	}
}

func NewNetOutRelatedEstablishedRule(subnet, overlayNetwork string) GenericRule {
	return GenericRule{
		Properties: []string{
			"-s", subnet,
			"!", "-d", overlayNetwork,
			"-m", "state", "--state", "RELATED,ESTABLISHED",
			"--jump", "RETURN",
		},
	}
}

func NewNetOutDefaultRejectRule(subnet, overlayNetwork string) GenericRule {
	return GenericRule{
		Properties: []string{
			"-s", subnet,
			"!", "-d", overlayNetwork,
			"--jump", "REJECT",
			"--reject-with", "icmp-port-unreachable",
		},
	}
}
