package poller

import (
	"net"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/runtimeschema/metric"
)

const netInterfaceCount = metric.Metric("NetInterfaceCount")

type SystemMetrics struct {
	Logger       lager.Logger
	PollInterval time.Duration
}

func (m *SystemMetrics) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)
	for {
		select {
		case <-signals:
			return nil
		case <-time.After(m.PollInterval):
			m.measure(m.Logger.Session("measure"))
		}
	}
}

func countNetworkInterfaces() (int, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return 0, err
	}
	return len(ifaces), nil
}

func (m *SystemMetrics) measure(logger lager.Logger) {
	nInterfaces, err := countNetworkInterfaces()
	if err != nil {
		logger.Error("count-network-interfaces", err)
		return
	}

	if err := netInterfaceCount.Send(nInterfaces); err != nil {
		logger.Error("failed-to-send-metric", err, lager.Data{
			"metric": netInterfaceCount})
		return
	}
}
