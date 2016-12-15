package planner

import (
	"lib/rules"
	"vxlan-policy-agent/enforcer"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/loggingStateGetter.go --fake-name LoggingStateGetter . loggingStateGetter
type loggingStateGetter interface {
	IsEnabled() bool
}

type VxlanDefaultLocalPlanner struct {
	Logger       lager.Logger
	LocalSubnet  string
	Chain        enforcer.Chain
	LoggingState loggingStateGetter
}

func (p *VxlanDefaultLocalPlanner) GetRulesAndChain() (enforcer.RulesWithChain, error) {
	theRules, err := p.GetRules()
	if err != nil {
		return enforcer.RulesWithChain{}, err
	}

	return enforcer.RulesWithChain{
		Chain: p.Chain,
		Rules: theRules,
	}, nil
}

func (p *VxlanDefaultLocalPlanner) GetRules() ([]rules.IPTablesRule, error) {
	ruleset := []rules.IPTablesRule{rules.NewAcceptExistingLocalRule()}

	if p.LoggingState.IsEnabled() {
		ruleset = append(ruleset, rules.NewLogLocalRejectRule(p.LocalSubnet))
	}

	ruleset = append(ruleset, rules.NewDefaultDenyLocalRule(p.LocalSubnet))

	return ruleset, nil
}

type VxlanDefaultRemotePlanner struct {
	Logger       lager.Logger
	VNI          int
	Chain        enforcer.Chain
	LoggingState loggingStateGetter
}

func (p *VxlanDefaultRemotePlanner) GetRulesAndChain() (enforcer.RulesWithChain, error) {
	theRules, err := p.GetRules()
	if err != nil {
		return enforcer.RulesWithChain{}, err
	}

	return enforcer.RulesWithChain{
		Chain: p.Chain,
		Rules: theRules,
	}, nil
}

func (p *VxlanDefaultRemotePlanner) GetRules() ([]rules.IPTablesRule, error) {
	ruleset := []rules.IPTablesRule{rules.NewAcceptExistingRemoteRule(p.VNI)}

	if p.LoggingState.IsEnabled() {
		ruleset = append(ruleset, rules.NewLogRemoteRejectRule(p.VNI))
	}

	ruleset = append(ruleset, rules.NewDefaultDenyRemoteRule(p.VNI))

	return ruleset, nil
}
