package pollers

import (
	"os"
	"time"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o ../fakes/timer.go --fake-name Timer . timer
type timer interface {
	After() <-chan time.Time
}

//go:generate counterfeiter -o ../fakes/converger.go --fake-name Converger . converger
type converger interface {
	Converge() error
}

type PeerPoller struct {
	Logger    lager.Logger
	Converger converger
	Timer     timer
}

func (p *PeerPoller) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	p.Logger.Info("starting")
	close(ready)
	for {
		select {
		case <-p.Timer.After():
			if err := p.Converger.Converge(); err != nil {
				p.Logger.Error("converge", err)
			}
		case <-signals:
			return nil
		}
	}
}
