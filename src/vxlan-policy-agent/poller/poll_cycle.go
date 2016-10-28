package poller

import (
	"fmt"
	"lib/rules"
	"time"
	"vxlan-policy-agent/agent_metrics"
)

//go:generate counterfeiter -o ../fakes/planner.go --fake-name Planner . planner
type planner interface {
	GetRules() (rules.RulesWithChain, error)
}

type SinglePollCycle struct {
	Planner planner

	Enforcer          rules.RuleEnforcer
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
