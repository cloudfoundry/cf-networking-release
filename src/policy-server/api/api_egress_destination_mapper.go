package api

import (
	"encoding/json"
	"fmt"
	"policy-server/store"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
)

type EgressDestinationMapper struct {
	Marshaler        marshal.Marshaler
	PayloadValidator egressDestinationsValidator
}

//go:generate counterfeiter -o fakes/egress_destinations_validator.go --fake-name EgressDestinationsValidator . egressDestinationsValidator
type egressDestinationsValidator interface {
	ValidateEgressDestinations([]EgressDestination) error
}

type DestinationsPayload struct {
	TotalDestinations  int                 `json:"total_destinations"`
	EgressDestinations []EgressDestination `json:"destinations"`
}

func (p *EgressDestinationMapper) AsBytes(egressDestinations []store.EgressDestination) ([]byte, error) {
	apiEgressDestinations := make([]EgressDestination, len(egressDestinations))

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

func (p *EgressDestinationMapper) AsEgressDestinations(egressDestinations []byte) ([]store.EgressDestination, error) {
	payload := &DestinationsPayload{}
	err := json.Unmarshal(egressDestinations, payload)
	if err != nil {
		return []store.EgressDestination{}, fmt.Errorf("unmarshal json: %s", err)
	}

	err = p.PayloadValidator.ValidateEgressDestinations(payload.EgressDestinations)
	if err != nil {
		return []store.EgressDestination{}, fmt.Errorf("validate destinations: %s", err)
	}

	storeEgressDestinations := make([]store.EgressDestination, len(payload.EgressDestinations))
	for i, apiDest := range payload.EgressDestinations {
		storeEgressDestinations[i] = apiDest.asStoreEgressDestination()
	}
	return storeEgressDestinations, nil
}

func asApiEgressDestination(storeEgressDestination store.EgressDestination) EgressDestination {
	apiEgressDestination := &EgressDestination{
		GUID:        storeEgressDestination.GUID,
		Name:        storeEgressDestination.Name,
		Description: storeEgressDestination.Description,
	}

	for _, rule := range storeEgressDestination.Rules {
		var ports []Ports

		if len(rule.Ports) > 0 {
			ports = []Ports{
				{
					Start: rule.Ports[0].Start,
					End:   rule.Ports[0].End,
				},
			}
		}
		var icmpType, icmpCode *int
		if rule.Protocol == "icmp" {
			icmpType = &rule.ICMPType
			icmpCode = &rule.ICMPCode
		}

		apiEgressDestination.Rules = append(apiEgressDestination.Rules, EgressDestinationRule{
			Protocol:    rule.Protocol,
			Ports:       ports,
			IPRanges:    fmt.Sprintf("%s-%s", rule.IPRanges[0].Start, rule.IPRanges[0].End),
			ICMPType:    icmpType,
			ICMPCode:    icmpCode,
			Description: rule.Description,
		})
	}

	return *apiEgressDestination
}

func (d *EgressDestination) asStoreEgressDestination() store.EgressDestination {
	destination := store.EgressDestination{
		GUID:        d.GUID,
		Name:        d.Name,
		Description: d.Description,
	}

	for _, rule := range d.Rules {
		ipRanges := []store.IPRange{}
		splitRange := strings.Split(rule.IPRanges, "-")
		ipRanges = append(ipRanges, store.IPRange{
			Start: splitRange[0],
			End:   splitRange[1],
		})

		ports := []store.Ports{}
		for _, apiPorts := range rule.Ports {
			ports = append(ports, store.Ports{
				Start: apiPorts.Start,
				End:   apiPorts.End,
			})
		}

		var icmpType, icmpCode int
		if rule.Protocol == "icmp" {
			if rule.ICMPType == nil {
				rule.ICMPType = &ICMPDefault
			}
			if rule.ICMPCode == nil {
				rule.ICMPCode = &ICMPDefault
			}
			icmpType = *rule.ICMPType
			icmpCode = *rule.ICMPCode
		}

		destination.Rules = append(destination.Rules, store.EgressDestinationRule{
			Protocol:    rule.Protocol,
			Ports:       ports,
			IPRanges:    ipRanges,
			ICMPType:    icmpType,
			ICMPCode:    icmpCode,
			Description: rule.Description,
		})
	}

	return destination
}
