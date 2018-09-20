package api

import (
	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"fmt"
	"policy-server/store"
)

type EgressPolicyMapper struct {
	Unmarshaler marshal.Unmarshaler
	Marshaler   marshal.Marshaler
}

type payload struct {
	TotalEgressPolicies int               `json:"total_egress_policies,omitempty"`
	EgressPolicies      []EgressPolicyPtr `json:"egress_policies,omitempty"`
}

type EgressPolicyPtr struct {
	ID          string                `json:"id"`
	Source      *EgressSource         `json:"source"`
	Destination *EgressDestinationPtr `json:"destination"`
}

type EgressDestinationPtr struct {
	GUID string `json:"id,omitempty"`
}

func (p *EgressPolicyMapper) AsBytes(storeEgressPolicies []store.EgressPolicy) ([]byte, error) {
	var apiEgressPolicyPtrs []EgressPolicyPtr
	for _, storeEgressPolicy := range storeEgressPolicies {
		apiEgressPolicyPtrs = append(apiEgressPolicyPtrs, asApiEgressPolicyPtr(storeEgressPolicy))
	}

	payload := &payload{
		TotalEgressPolicies: len(apiEgressPolicyPtrs),
		EgressPolicies:      apiEgressPolicyPtrs,
	}

	bytes, err := p.Marshaler.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %s", err)
	}

	return bytes, nil
}

func (p *EgressPolicyMapper) AsStoreEgressPolicy(bytes []byte) ([]store.EgressPolicy, error) {
	payload := &EgressPoliciesPayload{}
	err := p.Unmarshaler.Unmarshal(bytes, payload)
	if err != nil {
		return []store.EgressPolicy{}, fmt.Errorf("unmarshal json: %s", err)
	}

	var storeEgressPolicies []store.EgressPolicy
	for _, apiEgressPolicy := range payload.EgressPolicies {
		storeEgressPolicies = append(storeEgressPolicies, asStoreEgressPolicy(apiEgressPolicy))
	}

	return storeEgressPolicies, nil
}

func asApiEgressPolicyPtr(storeEgressPolicy store.EgressPolicy) EgressPolicyPtr {
	return EgressPolicyPtr{
		ID: storeEgressPolicy.ID,
		Destination: &EgressDestinationPtr{
			GUID: storeEgressPolicy.Destination.GUID,
		},
		Source: &EgressSource{
			ID:   storeEgressPolicy.Source.ID,
			Type: storeEgressPolicy.Source.Type,
		},
	}
}

func asStoreEgressPolicy(apiEgressPolicy EgressPolicy) store.EgressPolicy {
	return store.EgressPolicy{
		Destination: store.EgressDestination{
			GUID: apiEgressPolicy.Destination.GUID,
		},
		Source: store.EgressSource{
			ID:   apiEgressPolicy.Source.ID,
			Type: apiEgressPolicy.Source.Type,
		},
	}
}
