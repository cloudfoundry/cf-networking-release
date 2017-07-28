package api_v0_internal

import (
	"fmt"
	"policy-server/api"
	"policy-server/store"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
)

type policyMapper struct {
	Unmarshaler marshal.Unmarshaler
	Marshaler   marshal.Marshaler
}

func NewMapper(Unmarshaler marshal.Unmarshaler, Marshaler marshal.Marshaler) api.PolicyMapper {
	return &policyMapper{
		Unmarshaler: Unmarshaler,
		Marshaler:   Marshaler,
	}
}

func (p *policyMapper) AsStorePolicy(bytes []byte) ([]store.Policy, error) {
	// this function should never be used
	panic("as store policy was called for internal api")
}

func (p *policyMapper) AsBytes(storePolicies []store.Policy) ([]byte, error) {
	// convert store.Policy to api_v0_internal.Policy
	apiPolicies := []Policy{}
	for _, policy := range storePolicies {
		policyToAdd, canMap := mapStorePolicy(policy)
		if canMap {
			apiPolicies = append(apiPolicies, policyToAdd)
		}
	}

	// convert api_v0_internal.Policy payload to bytes
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
			Ports: Ports{
				Start: storePolicy.Destination.Ports.Start,
				End:   storePolicy.Destination.Ports.End,
			},
		},
	}, true
}
