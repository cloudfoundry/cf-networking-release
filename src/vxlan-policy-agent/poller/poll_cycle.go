package poller

import (
	"fmt"
	"time"
	"vxlan-policy-agent/agent_metrics"
	"vxlan-policy-agent/enforcer"
)

//go:generate counterfeiter -o ../fakes/planner.go --fake-name Planner . planner
type planner interface {
	GetRules() (enforcer.RulesWithChain, error)
}

//go:generate counterfeiter -o ../fakes/rule_enforcer.go --fake-name RuleEnforcer . ruleEnforcer
type ruleEnforcer interface {
	EnforceRulesAndChain(enforcer.RulesWithChain) error
}

type SinglePollCycle struct {
	Planner planner

	Enforcer          ruleEnforcer
	CollectionEmitter agent_metrics.TimeMetricsEmitter
}

func (m *SinglePollCycle) DoCycle() error {
	pollStartTime := time.Now()
	rulesWithChain, err := m.Planner.GetRules()
	if err != nil {
		return fmt.Errorf("get-rules: %s", err)
	}

	enforceStartTime := time.Now()
	err = m.Enforcer.EnforceRulesAndChain(rulesWithChain)
	if err != nil {
		return fmt.Errorf("enforce: %s", err)
	}
	enforceDuration := time.Now().Sub(enforceStartTime)
	pollDuration := time.Now().Sub(pollStartTime)
	m.CollectionEmitter.EmitAll(map[string]time.Duration{
		agent_metrics.MetricEnforceDuration: enforceDuration,
		agent_metrics.MetricPollDuration:    pollDuration,
	})

	return nil
}
