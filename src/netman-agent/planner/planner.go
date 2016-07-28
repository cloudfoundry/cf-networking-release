package planner

import (
	"fmt"
	"netman-agent/models"
	"netman-agent/rules"

	"code.cloudfoundry.org/lager"
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
	ruleEnforcer   rules.RuleEnforcer
}

func New(logger lager.Logger, storeReader storeReader, policyClient policyClient, vni int, localSubnet string, overlayNetwork string, ruleEnforcer rules.RuleEnforcer) *Planner {
	return &Planner{
		Logger:         logger,
		storeReader:    storeReader,
		policyClient:   policyClient,
		VNI:            vni,
		LocalSubnet:    localSubnet,
		OverlayNetwork: overlayNetwork,
		ruleEnforcer:   ruleEnforcer,
	}
}

func (p *Planner) DefaultLocalRules() error {
	ruleset := []rules.Rule{}

	ruleset = append(ruleset,
		rules.NewAcceptExistingLocalRule(),
		rules.NewLogRule(
			[]string{
				"-i", "cni-flannel0",
				"-s", p.LocalSubnet,
				"-d", p.LocalSubnet,
			},
			"DROP_LOCAL: ",
		),
		rules.NewDefaultDenyLocalRule(p.LocalSubnet),
	)

	return p.ruleEnforcer.Enforce("filter", "FORWARD", "netman--local-", ruleset)
}

func (p *Planner) DefaultRemoteRules() error {
	ruleset := []rules.Rule{}

	ruleset = append(ruleset,
		rules.NewAcceptExistingRemoteRule(p.VNI),
		rules.NewLogRule(
			[]string{"-i", fmt.Sprintf("flannel.%d", p.VNI)},
			"DROP_REMOTE: ",
		),
		rules.NewDefaultDenyRemoteRule(p.VNI),
	)

	return p.ruleEnforcer.Enforce("filter", "FORWARD", "netman--remote-", ruleset)
}

func (p *Planner) DefaultEgressRules() error {
	return p.ruleEnforcer.Enforce(
		"nat",
		"POSTROUTING",
		"netman--postrout-",
		[]rules.Rule{rules.NewDefaultEgressRule(p.LocalSubnet, p.OverlayNetwork)},
	)
}

func (p *Planner) Update() error {
	ruleset, err := p.Rules()
	if err != nil {
		return err
	}
	err = p.ruleEnforcer.Enforce("filter", "FORWARD", "netman--forward-", ruleset)
	if err != nil {
		return err
	}
	return nil
}

func (p *Planner) Rules() ([]rules.Rule, error) {
	containers := p.storeReader.GetContainers()
	policies, err := p.policyClient.GetPolicies()
	if err != nil {
		p.Logger.Error("get-policies", err)
		return nil, fmt.Errorf("get policies failed: %s", err)
	}

	ruleset := []rules.Rule{}

	for _, policy := range policies {
		srcContainers, srcOk := containers[policy.Source.ID]
		dstContainers, dstOk := containers[policy.Destination.ID]

		if dstOk {
			for _, dstContainer := range dstContainers {
				ruleset = append(
					ruleset,
					rules.NewRemoteAllowRule(
						p.VNI,
						dstContainer.IP,
						policy.Destination.Protocol,
						policy.Destination.Port,
						policy.Source.Tag,
						policy.Source.ID,
						policy.Destination.ID,
					),
				)
			}
		}

		if srcOk {
			for _, srcContainer := range srcContainers {
				ruleset = append(
					ruleset,
					rules.NewGBPTagRule(srcContainer.IP, policy.Source.Tag, policy.Source.ID),
				)
			}
		}

		if srcOk && dstOk {
			for _, srcContainer := range srcContainers {
				for _, dstContainer := range dstContainers {
					ruleset = append(
						ruleset,
						rules.NewLocalAllowRule(
							srcContainer.IP,
							dstContainer.IP,
							policy.Destination.Protocol,
							policy.Destination.Port,
							policy.Source.ID,
							policy.Destination.ID,
						),
					)
				}
			}
		}
	}

	return ruleset, nil
}
