package api

import (
	"fmt"
	"policy-server/store"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
)

type EgressPolicyMapper struct {
	Unmarshaler marshal.Unmarshaler
	Marshaler   marshal.Marshaler
	Validator   egressValidator
}

//go:generate counterfeiter -o fakes/egress_validator.go --fake-name EgressValidator . egressValidator
type egressValidator interface {
	ValidateEgressPolicies([]EgressPolicy) error
}

type payload struct {
	TotalEgressPolicies int            `json:"total_egress_policies"`
	EgressPolicies      []EgressPolicy `json:"egress_policies"`
}

func withPopulatedDestinations(storeEgressPolicy store.EgressPolicy) EgressPolicy {
	egressDestination := asApiEgressDestination(storeEgressPolicy.Destination)
	return EgressPolicy{
		ID:          storeEgressPolicy.ID,
		Destination: &egressDestination,
		Source: &EgressSource{
			ID:   storeEgressPolicy.Source.ID,
			Type: storeEgressPolicy.Source.Type,
		},
		AppLifecycle: storeEgressPolicy.AppLifecycle,
	}
}

func withDestinationPointer(storeEgressPolicy store.EgressPolicy) EgressPolicy {
	return EgressPolicy{
		ID: storeEgressPolicy.ID,
		Destination: &EgressDestination{
			GUID: storeEgressPolicy.Destination.GUID,
		},
		Source: &EgressSource{
			ID:   storeEgressPolicy.Source.ID,
			Type: storeEgressPolicy.Source.Type,
		},
		AppLifecycle: storeEgressPolicy.AppLifecycle,
	}
}

func (p *EgressPolicyMapper) AsBytesWithStrategy(storeEgressPolicies []store.EgressPolicy, mappingStrategy func(store.EgressPolicy) EgressPolicy) ([]byte, error) {
	apiEgressPolicies := make([]EgressPolicy, len(storeEgressPolicies))
	for i, storeEgressPolicy := range storeEgressPolicies {
		apiEgressPolicies[i] = mappingStrategy(storeEgressPolicy)
	}

	payload := &payload{
		TotalEgressPolicies: len(apiEgressPolicies),
		EgressPolicies:      apiEgressPolicies,
	}

	bytes, err := p.Marshaler.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %s", err)
	}

	return bytes, nil
}

func (p *EgressPolicyMapper) AsBytesWithPopulatedDestinations(storeEgressPolicies []store.EgressPolicy) ([]byte, error) {
	return p.AsBytesWithStrategy(storeEgressPolicies, withPopulatedDestinations)
}

func (p *EgressPolicyMapper) AsBytes(storeEgressPolicies []store.EgressPolicy) ([]byte, error) {
	return p.AsBytesWithStrategy(storeEgressPolicies, withDestinationPointer)
}

func (p *EgressPolicyMapper) AsStoreEgressPolicy(bytes []byte) ([]store.EgressPolicy, error) {
	payload := &EgressPoliciesPayload{}
	err := p.Unmarshaler.Unmarshal(bytes, payload)
	if err != nil {
		return []store.EgressPolicy{}, fmt.Errorf("unmarshal json: %s", err)
	}

	err = p.Validator.ValidateEgressPolicies(payload.EgressPolicies)
	if err != nil {
		return []store.EgressPolicy{}, fmt.Errorf("validating egress policies: %s", err)
	}

	var storeEgressPolicies []store.EgressPolicy
	for _, apiEgressPolicy := range payload.EgressPolicies {
		storeEgressPolicies = append(storeEgressPolicies, asStoreEgressPolicy(apiEgressPolicy))
	}

	return storeEgressPolicies, nil
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
		AppLifecycle: apiEgressPolicy.AppLifecycle,
	}
}
