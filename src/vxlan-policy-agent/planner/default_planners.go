package planner

import (
	"fmt"
	"lib/rules"

	"code.cloudfoundry.org/lager"
)

type VxlanDefaultLocalPlanner struct {
	Logger      lager.Logger
	LocalSubnet string
	Chain       rules.Chain
}

func (p *VxlanDefaultLocalPlanner) GetRulesAndChain() (rules.RulesWithChain, error) {
	theRules, err := p.GetRules()
	if err != nil {
		return rules.RulesWithChain{}, err
	}

	return rules.RulesWithChain{
		Chain: p.Chain,
		Rules: theRules,
	}, nil
}

func (p *VxlanDefaultLocalPlanner) GetRules() ([]rules.Rule, error) {
	ruleset := []rules.Rule{}

	ruleset = append(ruleset,
		rules.NewAcceptExistingLocalRule(),
		rules.NewLogRule(
			[]string{
				"-i", "cni-flannel0",
				"-s", p.LocalSubnet,
				"-d", p.LocalSubnet,
			},
			"REJECT_LOCAL: ",
		),
		rules.NewDefaultDenyLocalRule(p.LocalSubnet),
	)

	return ruleset, nil
}

type VxlanDefaultRemotePlanner struct {
	Logger lager.Logger
	VNI    int
	Chain  rules.Chain
}

func (p *VxlanDefaultRemotePlanner) GetRulesAndChain() (rules.RulesWithChain, error) {
	theRules, err := p.GetRules()
	if err != nil {
		return rules.RulesWithChain{}, err
	}

	return rules.RulesWithChain{
		Chain: p.Chain,
		Rules: theRules,
	}, nil
}

func (p *VxlanDefaultRemotePlanner) GetRules() ([]rules.Rule, error) {
	ruleset := []rules.Rule{}

	ruleset = append(ruleset,
		rules.NewAcceptExistingRemoteRule(p.VNI),
		rules.NewLogRule(
			[]string{"-i", fmt.Sprintf("flannel.%d", p.VNI)},
			"REJECT_REMOTE: ",
		),
		rules.NewDefaultDenyRemoteRule(p.VNI),
	)

	return ruleset, nil
}

type VxlanDefaultMasqueradePlanner struct {
	Logger         lager.Logger
	LocalSubnet    string
	OverlayNetwork string
	Chain          rules.Chain
}

func (p *VxlanDefaultMasqueradePlanner) GetRules() ([]rules.Rule, error) {
	ruleset := []rules.Rule{}

	ruleset = append(ruleset,
		rules.NewDefaultEgressRule(p.LocalSubnet, p.OverlayNetwork),
	)

	return ruleset, nil
}

func (p *VxlanDefaultMasqueradePlanner) GetRulesAndChain() (rules.RulesWithChain, error) {
	theRules, err := p.GetRules()
	if err != nil {
		return rules.RulesWithChain{}, err
	}

	return rules.RulesWithChain{
		Chain: p.Chain,
		Rules: theRules,
	}, nil
}
