package poller

import (
	"os"
	"time"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
)

type Poller struct {
	Logger       lager.Logger
	PollInterval time.Duration
	GardenClient garden.Client
}

func (m *Poller) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)
	for {
		select {
		case <-signals:
			return nil
		case <-time.After(m.PollInterval):
			m.pollOnce()
		}
	}
}

func (m *Poller) pollOnce() {
	logger := m.Logger.Session("pollOnce")
	logger.Info("start")
	defer logger.Info("done")

	properties := garden.Properties{}
	allContainers, err := m.GardenClient.Containers(properties)
	if err != nil {
		logger.Error("gardenClient.Containers", err)
		return
	}

	for _, container := range allContainers {
		info, err := container.Info()
		if err != nil {
			logger.Error("container.Info", err)
			return
		}
		for _, mapping := range info.MappedPorts {
			logger.Info("net-in-rule", lager.Data{
				"host_port":      mapping.HostPort,
				"container_port": mapping.ContainerPort,
				"container_ip":   info.ContainerIP,
				"external_ip":    info.ExternalIP,
			})
		}
	}
}
