package metrics

import (
	"os"
	"time"

	dropsondemetrics "github.com/cloudfoundry/dropsonde/metrics"
)

type Uptime struct {
	interval time.Duration
	started  int64
}

func NewUptime(interval time.Duration) *Uptime {
	return &Uptime{
		interval: interval,
		started:  time.Now().Unix(),
	}
}

func (u *Uptime) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)
	ticker := time.NewTicker(u.interval)

	for {
		select {
		case <-ticker.C:
			dropsondemetrics.SendValue("uptime", float64(time.Now().Unix()-u.started), "seconds")
		case <-signals:
			ticker.Stop()
			return nil
		}
	}
}
