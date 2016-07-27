package rules

import (
	"fmt"
	"netman-agent/models"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/store_reader.go --fake-name StoreReader . storeReader
type storeReader interface {
	GetContainers() map[string][]models.Container
}

//go:generate counterfeiter -o ../fakes/policy_client.go --fake-name PolicyClient . policyClient
type policyClient interface {
	GetPolicies() ([]models.Policy, error)
}

type Planner struct {
	Logger         lager.Logger
	storeReader    storeReader
	policyClient   policyClient
	VNI            int
	LocalSubnet    string
	OverlayNetwork string
	RuleEnforcer   RuleEnforcer
}

//go:generate counterfeiter -o ../fakes/rule_enforcer.go --fake-name RuleEnforcer . RuleEnforcer
type RuleEnforcer interface {
	Enforce(table, parentChain, chain string, r []Rule) error
}

func New(logger lager.Logger, storeReader storeReader, policyClient policyClient, vni int, localSubnet string, overlayNetwork string, ruleEnforcer RuleEnforcer) *Planner {
	return &Planner{
		Logger:         logger,
		storeReader:    storeReader,
		policyClient:   policyClient,
		VNI:            vni,
		LocalSubnet:    localSubnet,
		OverlayNetwork: overlayNetwork,
		RuleEnforcer:   ruleEnforcer,
	}
}

func (p *Planner) DefaultLocalRules() []Rule {
	rules := []Rule{}

	rules = append(rules, GenericRule{
		Properties: []string{
			"-i", "cni-flannel0",
			"-m", "state", "--state", "ESTABLISHED,RELATED",
			"-j", "ACCEPT",
		},
	}, GenericRule{
		Properties: []string{
			"-i", "cni-flannel0",
			"-s", p.LocalSubnet,
			"-d", p.LocalSubnet,
			"-m", "limit", "--limit", "2/min",
			"-j", "LOG",
			"--log-prefix", "DROP_LOCAL: ",
		},
	}, GenericRule{
		Properties: []string{
			"-i", "cni-flannel0",
			"-s", p.LocalSubnet,
			"-d", p.LocalSubnet,
			"-j", "DROP",
		},
	})

	return rules
}

func (p *Planner) DefaultRemoteRules() []Rule {
	rules := []Rule{}

	rules = append(rules, GenericRule{
		Properties: []string{
			"-i", fmt.Sprintf("flannel.%d", p.VNI),
			"-m", "state", "--state", "ESTABLISHED,RELATED",
			"-j", "ACCEPT",
		},
	}, GenericRule{
		Properties: []string{
			"-i", fmt.Sprintf("flannel.%d", p.VNI),
			"-m", "limit", "--limit", "2/min",
			"-j", "LOG",
			"--log-prefix", "DROP_REMOTE: ",
		},
	}, GenericRule{
		Properties: []string{
			"-i", fmt.Sprintf("flannel.%d", p.VNI),
			"-j", "DROP",
		},
	})

	return rules
}

func (p *Planner) DefaultEgressRules() []Rule {
	return []Rule{
		NewDefaultEgressRule(p.LocalSubnet, p.OverlayNetwork),
	}
}

func (p *Planner) Update() error {
	rules, err := p.Rules()
	if err != nil {
		return err
	}
	err = p.RuleEnforcer.Enforce("filter", "FORWARD", "netman--forward-", rules)
	if err != nil {
		return err
	}
	return nil
}

func (p *Planner) Rules() ([]Rule, error) {
	containers := p.storeReader.GetContainers()
	policies, err := p.policyClient.GetPolicies()
	if err != nil {
		p.Logger.Error("get-policies", err)
		return nil, fmt.Errorf("get policies failed: %s", err)
	}

	rules := []Rule{}

	for _, policy := range policies {
		srcContainers, srcOk := containers[policy.Source.ID]
		dstContainers, dstOk := containers[policy.Destination.ID]

		if dstOk {
			for _, dstContainer := range dstContainers {
				rules = append(
					rules,
					NewRemoteAllowRule(
						p.VNI,
						dstContainer.IP,
						policy.Destination.Protocol,
						policy.Destination.Port,
						policy.Source.Tag,
					),
				)
			}
		}

		if srcOk {
			for _, srcContainer := range srcContainers {
				rules = append(
					rules,
					NewGBPTagRule(srcContainer.IP, policy.Source.Tag),
				)
			}
		}

		if srcOk && dstOk {
			for _, srcContainer := range srcContainers {
				for _, dstContainer := range dstContainers {
					rules = append(
						rules,
						NewLocalAllowRule(
							srcContainer.IP,
							dstContainer.IP,
							policy.Destination.Protocol,
							policy.Destination.Port,
						),
					)
				}
			}
		}
	}

	return rules, nil
}
