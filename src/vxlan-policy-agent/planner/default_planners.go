package planner

import (
	"fmt"
	"netman-agent/rules"

	"code.cloudfoundry.org/lager"
)

type VxlanDefaultLocalPlanner struct {
	Logger      lager.Logger
	LocalSubnet string
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
