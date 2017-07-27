package api

import (
	"fmt"
	"policy-server/store"

	"code.cloudfoundry.org/cf-networking-helpers/marshal"
)

type policyMapper struct {
	Unmarshaler marshal.Unmarshaler
	Marshaler   marshal.Marshaler
}

func NewMapper(Unmarshaler marshal.Unmarshaler, Marshaler marshal.Marshaler) PolicyMapper {
	return &policyMapper{
		Unmarshaler: Unmarshaler,
		Marshaler:   Marshaler,
	}
}

func (p *policyMapper) AsStorePolicy(bytes []byte) ([]store.Policy, error) {
	payload := &Policies{}
	err := p.Unmarshaler.Unmarshal(bytes, payload)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json: %s", err)
	}

	storePolicies := []store.Policy{}
	for _, policy := range payload.Policies {
		storePolicies = append(storePolicies, policy.asStorePolicy())
	}
	return storePolicies, nil
}
func (p *policyMapper) AsBytes(storePolicies []store.Policy) ([]byte, error) {
	// convert store.Policy to api.Policy
	apiPolicies := []Policy{}
	for _, policy := range storePolicies {
		apiPolicies = append(apiPolicies, mapStorePolicy(policy))
	}

	// convert api.Policy payload to bytes
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
		ID:  tag.ID,
		Tag: tag.Tag,
	}
}

func MapStoreTags(tags []store.Tag) []Tag {
	apiTags := []Tag{}

	for _, tag := range tags {
		apiTags = append(apiTags, MapStoreTag(tag))
	}
	return apiTags
}
