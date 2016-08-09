package planner

import (
	"encoding/json"
	"netman-agent/rules"

	"code.cloudfoundry.org/garden"
)

type NetOutPlanner struct {
	GardenClient   garden.Client
	OverlayNetwork string
}

func (netOutPlanner *NetOutPlanner) GetRules() ([]rules.Rule, error) {
	properties := garden.Properties{}
	allContainers, err := netOutPlanner.GardenClient.Containers(properties)
	if err != nil {
		return nil, err
	}

	specs := []rules.Rule{}
	for _, container := range allContainers {
		info, err := container.Info()
		if err != nil {
			return nil, err
		}

		var netOuts []garden.NetOutRule
		err = json.Unmarshal([]byte(info.Properties["network.external-networker.net-out"]), &netOuts)
		if err != nil {
			return nil, err
		}

		for _, netOut := range netOuts {
			for _, ipRange := range netOut.Networks {
				if len(netOut.Ports) > 0 && udpOrTcp(netOut.Protocol) {
					for _, portRange := range netOut.Ports {
						specs = append(specs, rules.NewNetOutWithPortsRule(
							info.ContainerIP,
							ipRange.Start.String(),
							ipRange.End.String(),
							int(portRange.Start),
							int(portRange.End),
							protocolToString(netOut.Protocol),
							info.Properties["network.app_id"],
						))
					}
				} else {
					specs = append(specs, rules.NewNetOutRule(
						info.ContainerIP,
						ipRange.Start.String(),
						ipRange.End.String(),
						info.Properties["network.app_id"],
					))
				}
			}
		}
	}

	specs = append(specs, rules.NewNetOutDefaultRejectRule(netOutPlanner.OverlayNetwork))

	return specs, nil
}

func udpOrTcp(protocol garden.Protocol) bool {
	return protocol == garden.ProtocolTCP || protocol == garden.ProtocolUDP
}

func protocolToString(protocol garden.Protocol) string {
	switch protocol {
	case garden.ProtocolTCP:
		return "tcp"
	case garden.ProtocolUDP:
		return "udp"
	default:
		return "all"
	}
}
