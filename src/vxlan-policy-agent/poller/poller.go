package poller

import (
	"lib/rules"
	"os"
	"time"
	"vxlan-policy-agent/agent_metrics"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/planner.go --fake-name Planner . planner
type planner interface {
	GetRules() (rules.RulesWithChain, error)
}

type Poller struct {
	Logger       lager.Logger
	PollInterval time.Duration
	Planner      planner

	Enforcer          rules.RuleEnforcer
	CollectionEmitter agent_metrics.TimeMetricsEmitter
}

func (m *Poller) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	for {
		select {
		case <-signals:
			return nil
		case <-time.After(m.PollInterval):
			pollStartTime := time.Now()
			rulesWithChain, err := m.Planner.GetRules()
			if err != nil {
				m.Logger.Error("get-rules", err)
				continue
			}

			enforceStartTime := time.Now()
			err = m.Enforcer.EnforceRulesAndChain(rulesWithChain)
			if err != nil {
				m.Logger.Error("enforce", err)
				continue
			}
			enforceDuration := time.Now().Sub(enforceStartTime)
			pollDuration := time.Now().Sub(pollStartTime)
			m.CollectionEmitter.EmitAll(map[string]time.Duration{
				agent_metrics.MetricEnforceDuration: enforceDuration,
				agent_metrics.MetricPollDuration:    pollDuration,
			})
		}
	}
}
