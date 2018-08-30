package api

import (
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"fmt"
	"policy-server/store"
)

type EgressDestinationMapper struct {
	Marshaler marshal.Marshaler
}

type DestinationsPayload struct {
	TotalDestinations  int                  `json:"total_destinations"`
	EgressDestinations []*EgressDestination `json:"destinations"`
}

func (p *EgressDestinationMapper) AsBytes(egressDestinations []store.EgressDestination) ([]byte, error) {
	apiEgressDestinations := make([]*EgressDestination, len(egressDestinations))

	for i, storeEgressDestination := range egressDestinations {
		apiEgressDestinations[i] = asApiEgressDestination(storeEgressDestination)
	}

	payload := &DestinationsPayload{
		TotalDestinations:  len(apiEgressDestinations),
		EgressDestinations: apiEgressDestinations,
	}

	bytes, err := p.Marshaler.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %s", err)
	}
	return bytes, nil
}

func asApiEgressDestination(storeEgressDestination store.EgressDestination) *EgressDestination {
	var ports []Ports

	if len(storeEgressDestination.Ports) > 0 {
		ports = []Ports{
			{
				Start: storeEgressDestination.Ports[0].Start,
				End:   storeEgressDestination.Ports[0].End,
			},
		}
	}

	firstIPRange := storeEgressDestination.IPRanges[0]

	apiEgressDestination := &EgressDestination{
		GUID:     storeEgressDestination.ID,
		Protocol: storeEgressDestination.Protocol,
		Ports:    ports,
		IPRanges: []IPRange{{
			Start: firstIPRange.Start,
			End:   firstIPRange.End,
		}},
	}

	if storeEgressDestination.Protocol == "icmp" {
		apiEgressDestination.ICMPType = &storeEgressDestination.ICMPType
		apiEgressDestination.ICMPCode = &storeEgressDestination.ICMPCode
	}
	return apiEgressDestination
}
