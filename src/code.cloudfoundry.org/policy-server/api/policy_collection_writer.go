package api

import (
	"fmt"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
	"code.cloudfoundry.org/policy-server/store"
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

	apiEgressPolicies := []InternalEgressPolicy{}
	for _, egressPolicy := range egressPolicies {
		withoutRulesPolicies := mapStoreEgressPolicyToInternalAPIEgressPolicy(egressPolicy)
		apiEgressPolicies = append(apiEgressPolicies, withoutRulesPolicies...)
	}

	policyCollection := PolicyCollectionPayload{
		TotalPolicies:       len(apiPolicies),
		Policies:            apiPolicies,
		TotalEgressPolicies: len(apiEgressPolicies),
		EgressPolicies:      apiEgressPolicies,
	}
	bytes, err := p.Marshaler.Marshal(policyCollection)
	if err != nil {
		return []byte{}, fmt.Errorf("marshal json: %s", err)
	}

	return bytes, nil
}

func mapStoreEgressPolicyToInternalAPIEgressPolicy(storeEgressPolicy store.EgressPolicy) []InternalEgressPolicy {
	var policies []InternalEgressPolicy

	for _, rule := range storeEgressPolicy.Destination.Rules {
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

		firstIPRange := rule.IPRanges[0]

		policies = append(policies, InternalEgressPolicy{
			Source: &EgressSource{
				ID:   storeEgressPolicy.Source.ID,
				Type: storeEgressPolicy.Source.Type,
			},
			Destination: &InternalEgressDestination{
				GUID:        storeEgressPolicy.Destination.GUID,
				Name:        storeEgressPolicy.Destination.Name,
				Description: storeEgressPolicy.Destination.Description,
				Protocol:    rule.Protocol,
				Ports:       ports,
				IPRanges: []IPRange{{
					Start: firstIPRange.Start,
					End:   firstIPRange.End,
				}},
				ICMPType: icmpType,
				ICMPCode: icmpCode,
			},
			AppLifecycle: &storeEgressPolicy.AppLifecycle,
		})
	}
	return policies
}
