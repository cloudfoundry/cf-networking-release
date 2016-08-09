package poller

import (
	"netman-agent/rules"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/planner.go --fake-name Planner . planner
type planner interface {
	GetRules() ([]rules.Rule, error)
}

type Poller struct {
	Logger       lager.Logger
	PollInterval time.Duration
	Planner      planner

	Chain    rules.Chain
	Enforcer rules.RuleEnforcer
}

func (m *Poller) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)
	for {
		select {
		case <-signals:
			return nil
		case <-time.After(m.PollInterval):
			ruleset, err := m.Planner.GetRules()
			if err != nil {
				m.Logger.Error("get-rules", err)
				continue
			}

			err = m.Enforcer.EnforceOnChain(m.Chain, ruleset)
			if err != nil {
				m.Logger.Error("enforce", err)
				continue
			}
		}
	}
}
