package api_v0

import (
	"fmt"
	"policy-server/api"
	"policy-server/store"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
)

type policyMapper struct {
	Unmarshaler marshal.Unmarshaler
	Marshaler   marshal.Marshaler
	Validator   validator
}

func NewMapper(unmarshaler marshal.Unmarshaler, marshaler marshal.Marshaler, validator validator) api.PolicyMapper {
	return &policyMapper{
		Unmarshaler: unmarshaler,
		Marshaler:   marshaler,
		Validator:   validator,
	}
}

func (p *policyMapper) AsStorePolicy(bytes []byte) ([]store.Policy, error) {
	payload := &Policies{}
	err := p.Unmarshaler.Unmarshal(bytes, payload)
	if err != nil {
		return []store.Policy{}, fmt.Errorf("unmarshal json: %s", err)
	}

	err = p.Validator.ValidatePolicies(payload.Policies)
	if err != nil {
		return []store.Policy{}, fmt.Errorf("validate policies: %s", err)
	}

	storePolicies := []store.Policy{}
	for _, policy := range payload.Policies {
		storePolicies = append(storePolicies, policy.asStorePolicy())
	}

	return storePolicies, nil
}

func (p *policyMapper) AsBytes(storePolicies []store.Policy) ([]byte, error) {
	// convert store.Policy to api_v0.Policy
	apiPolicies := []Policy{}
	for _, policy := range storePolicies {
		policyToAdd, canMap := mapStorePolicy(policy)
		if canMap {
			apiPolicies = append(apiPolicies, policyToAdd)
		}
	}

	// convert api_v0.Policy payload to bytes
	payload := &Policies{
		TotalPolicies: len(apiPolicies),
		Policies:      apiPolicies,
	}
	bytes, err := p.Marshaler.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %s", err)
	}
	return bytes, nil
}

func (p *Policy) asStorePolicy() store.Policy {
	return store.Policy{
		Source: store.Source{
			ID:  p.Source.ID,
			Tag: p.Source.Tag,
		},
		Destination: store.Destination{
			ID:       p.Destination.ID,
			Tag:      p.Destination.Tag,
			Protocol: p.Destination.Protocol,
			Port:     p.Destination.Port,
			Ports: store.Ports{
				Start: p.Destination.Port,
				End:   p.Destination.Port,
			},
		},
	}
}

func mapStorePolicy(storePolicy store.Policy) (Policy, bool) {
	if storePolicy.Destination.Ports.Start != storePolicy.Destination.Ports.End {
		return Policy{}, false
	}
	return Policy{
		Source: Source{
			ID:  storePolicy.Source.ID,
			Tag: storePolicy.Source.Tag,
		},
		Destination: Destination{
			ID:       storePolicy.Destination.ID,
			Tag:      storePolicy.Destination.Tag,
			Protocol: storePolicy.Destination.Protocol,
			Port:     storePolicy.Destination.Ports.Start,
		},
	}, true
}
