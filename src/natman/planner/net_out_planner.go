package planner

import (
	"encoding/json"
	"lib/rules"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
)

type NetOutPlanner struct {
	GardenClient   garden.Client
	OverlayNetwork string
	Logger         lager.Logger
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
			netOutPlanner.Logger.Error("container-info", err, lager.Data{"info": info})
			return nil, err
		}

		var netOuts []garden.NetOutRule
		netoutJSON, ok := info.Properties["network.external-networker.net-out"]
		if !ok || netoutJSON == "" {
			continue
		}
		err = json.Unmarshal([]byte(netoutJSON), &netOuts)
		if err != nil {
			netOutPlanner.Logger.Error("netout-unmarshal-json", err, lager.Data{"properties": info.Properties["network.external-networker.net-out"]})
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

	specs = append(specs, rules.NewNetOutRelatedEstablishedRule(netOutPlanner.OverlayNetwork))
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
