package poller

import (
	"os"
	"time"

	"code.cloudfoundry.org/lager"
)

type Poller struct {
	Logger       lager.Logger
	PollInterval time.Duration

	SingleCycleFunc func() error
}

func (m *Poller) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	for {
		select {
		case <-signals:
			return nil
		case <-time.After(m.PollInterval):
			if err := m.SingleCycleFunc(); err != nil {
				m.Logger.Error("poll-cycle", err)
				continue
			}
		}
	}
}
