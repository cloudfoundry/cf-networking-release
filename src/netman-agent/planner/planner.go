package planner

import "netman-agent/rules"

type Planner struct {
	LocalSubnet    string
	OverlayNetwork string
	RuleEnforcer   rules.RuleEnforcer
}

func (p *Planner) DefaultEgressRules() error {
	return p.RuleEnforcer.Enforce(
		"nat",
		"POSTROUTING",
		"netman--postrout-",
		[]rules.Rule{rules.NewDefaultEgressRule(p.LocalSubnet, p.OverlayNetwork)},
	)
}
