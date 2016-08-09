package planner

import (
	"netman-agent/rules"

	"code.cloudfoundry.org/garden"
)

type NetInPlanner struct {
	GardenClient garden.Client
}

func (netInPlanner *NetInPlanner) GetRules() ([]rules.Rule, error) {
	properties := garden.Properties{}
	allContainers, err := netInPlanner.GardenClient.Containers(properties)
	if err != nil {
		return nil, err
	}

	specs := []rules.Rule{}
	for _, container := range allContainers {
		info, err := container.Info()
		if err != nil {
			return nil, err
		}
		for _, mapping := range info.MappedPorts {
			specs = append(specs, rules.NewNetInRule(
				info.ContainerIP,
				int(mapping.ContainerPort),
				info.ExternalIP,
				int(mapping.HostPort),
				info.Properties["network.app_id"]),
			)
		}
	}
	return specs, nil
}
