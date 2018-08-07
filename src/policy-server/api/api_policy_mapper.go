package api

import (
	"fmt"
	"policy-server/store"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
)

type policyMapper struct {
	Unmarshaler      marshal.Unmarshaler
	Marshaler        marshal.Marshaler
	PayloadValidator payloadValidator
}

func NewMapper(unmarshaler marshal.Unmarshaler, marshaler marshal.Marshaler, payloadValidator payloadValidator) PolicyMapper {
	return &policyMapper{
		Unmarshaler:      unmarshaler,
		Marshaler:        marshaler,
		PayloadValidator: payloadValidator,
	}
}

func (p *policyMapper) AsStorePolicy(bytes []byte) (store.PolicyCollection, error) {
	payload := &PoliciesPayload{}
	err := p.Unmarshaler.Unmarshal(bytes, payload)
	if err != nil {
		return store.PolicyCollection{}, fmt.Errorf("unmarshal json: %s", err)
	}

	err = p.PayloadValidator.ValidatePayload(payload)
	if err != nil {
		return store.PolicyCollection{}, fmt.Errorf("validate policies: %s", err)
	}

	var storePolicies []store.Policy
	for _, policy := range payload.Policies {
		storePolicies = append(storePolicies, policy.asStorePolicy())
	}

	var storeEgressPolicies []store.EgressPolicy
	for _, egressPolicy := range payload.EgressPolicies {
		storeEgressPolicies = append(storeEgressPolicies, egressPolicy.asStoreEgressPolicy())
	}

	return store.PolicyCollection{
		Policies:       storePolicies,
		EgressPolicies: storeEgressPolicies,
	}, nil
}

func (p *policyMapper) AsBytes(storePolicies []store.Policy, storeEgressPolicies []store.EgressPolicy) ([]byte, error) {
	// convert store.Policy to api.Policy
	apiPolicies := make([]Policy, len(storePolicies))
	for i, policy := range storePolicies {
		apiPolicies[i] = mapStorePolicy(policy)
	}

	apiEgressPolicies := make([]EgressPolicy, len(storeEgressPolicies))
	for i, egressPolicy := range storeEgressPolicies {
		apiEgressPolicies[i] = mapStoreEgressPolicy(egressPolicy)
	}

	// convert api.Policy payload to bytes
	payload := &PoliciesPayload{
		TotalPolicies:       len(apiPolicies),
		Policies:            apiPolicies,
		TotalEgressPolicies: len(apiEgressPolicies),
		EgressPolicies:      apiEgressPolicies,
	}
	bytes, err := p.Marshaler.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %s", err)
	}
	return bytes, nil
}

func (p *EgressPolicy) asStoreEgressPolicy() store.EgressPolicy {
	ipRanges := []store.IPRange{}
	for _, apiIPRange := range p.Destination.IPRanges {
		ipRanges = append(ipRanges, store.IPRange{
			Start: apiIPRange.Start,
			End:   apiIPRange.End,
		})
	}
	ports := []store.Ports{}
	for _, apiPorts := range p.Destination.Ports {
		ports = append(ports, store.Ports{
			Start: apiPorts.Start,
			End:   apiPorts.End,
		})
	}

	egressPolicy := store.EgressPolicy{
		Source: store.EgressSource{
			ID:   p.Source.ID,
			Type: p.Source.Type,
		},
		Destination: store.EgressDestination{
			Protocol: p.Destination.Protocol,
			Ports:    ports,
			IPRanges: ipRanges,
		},
	}

	if p.Destination.Protocol == "icmp" {
		egressPolicy.Destination.ICMPType = *p.Destination.ICMPType
		egressPolicy.Destination.ICMPCode = *p.Destination.ICMPCode
	}

	return egressPolicy
}

func (p *Policy) asStorePolicy() store.Policy {
	port := 0
	if p.Destination.Ports.Start == p.Destination.Ports.End {
		port = p.Destination.Ports.Start
	}
	return store.Policy{
		Source: store.Source{
			ID:  p.Source.ID,
			Tag: p.Source.Tag,
		},
		Destination: store.Destination{
			ID:       p.Destination.ID,
			Tag:      p.Destination.Tag,
			Protocol: p.Destination.Protocol,
			Port:     port,
			Ports: store.Ports{
				Start: p.Destination.Ports.Start,
				End:   p.Destination.Ports.End,
			},
		},
	}
}
func mapStoreEgressPolicy(storeEgressPolicy store.EgressPolicy) EgressPolicy {
	firstIPRange := storeEgressPolicy.Destination.IPRanges[0]

	var ports []Ports
	if len(storeEgressPolicy.Destination.Ports) > 0 {
		ports = []Ports{
			{
				Start: storeEgressPolicy.Destination.Ports[0].Start,
				End:   storeEgressPolicy.Destination.Ports[0].End,
			},
		}
	}

	egressPolicy := EgressPolicy{
		Source: &EgressSource{
			ID:   storeEgressPolicy.Source.ID,
			Type: storeEgressPolicy.Source.Type,
		},
		Destination: &EgressDestination{
			Protocol: storeEgressPolicy.Destination.Protocol,
			Ports:    ports,
			IPRanges: []IPRange{{
				Start: firstIPRange.Start,
				End:   firstIPRange.End,
			}},
		},
	}

	if storeEgressPolicy.Destination.Protocol == "icmp" {
		egressPolicy.Destination.ICMPType = &storeEgressPolicy.Destination.ICMPType
		egressPolicy.Destination.ICMPCode = &storeEgressPolicy.Destination.ICMPCode
	}

	return egressPolicy
}

func mapStorePolicy(storePolicy store.Policy) Policy {
	return Policy{
		Source: Source{
			ID:  storePolicy.Source.ID,
			Tag: storePolicy.Source.Tag,
		},
		Destination: Destination{
			ID:       storePolicy.Destination.ID,
			Tag:      storePolicy.Destination.Tag,
			Protocol: storePolicy.Destination.Protocol,
			Ports: Ports{
				Start: storePolicy.Destination.Ports.Start,
				End:   storePolicy.Destination.Ports.End,
			},
		},
	}
}

func MapStoreTag(tag store.Tag) Tag {
	return Tag{
		ID:   tag.ID,
		Tag:  tag.Tag,
		Type: tag.Type,
	}
}

func MapStoreTags(tags []store.Tag) []Tag {
	apiTags := []Tag{}

	for _, tag := range tags {
		apiTags = append(apiTags, MapStoreTag(tag))
	}
	return apiTags
}
