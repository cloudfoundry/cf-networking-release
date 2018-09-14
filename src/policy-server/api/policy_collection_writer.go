package api

import (
	"fmt"
	"policy-server/store"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
)

type policyCollectionWriter struct {
	Marshaler marshal.Marshaler
}

type PolicyCollection struct {
	TotalPolicies       int
	TotalEgressPolicies int
	Policies            []Policy
	EgressPolicies      []EgressPolicy
}

func NewPolicyCollectionWriter(marshaler marshal.Marshaler) PolicyCollectionWriter {
	return &policyCollectionWriter{
		Marshaler: marshaler,
	}
}

func (p *policyCollectionWriter) AsBytes(policies []store.Policy, egressPolicies []store.EgressPolicy) ([]byte, error) {
	apiPolicies := []Policy{}
	for _, policy := range policies {
		apiPolicies = append(apiPolicies, mapStorePolicy(policy))
	}

	apiEgressPolicies := []EgressPolicy{}
	for _, egressPolicy := range egressPolicies {
		apiEgressPolicies = append(apiEgressPolicies, mapStoreEgressPolicy(egressPolicy))
	}

	policyCollection := PolicyCollectionPayload{
		TotalPolicies:       len(policies),
		Policies:            apiPolicies,
		TotalEgressPolicies: len(egressPolicies),
		EgressPolicies:      apiEgressPolicies,
	}
	bytes, err := p.Marshaler.Marshal(policyCollection)
	if err != nil {
		return []byte{}, fmt.Errorf("marshal json: %s", err)
	}

	return bytes, nil
}

func mapStoreEgressPolicy(storeEgressPolicy store.EgressPolicy) EgressPolicy {
	destination := asApiEgressDestination(storeEgressPolicy.Destination)
	return EgressPolicy{
		Source: &EgressSource{
			ID:   storeEgressPolicy.Source.ID,
			Type: storeEgressPolicy.Source.Type,
		},
		Destination: &destination,
	}
}
