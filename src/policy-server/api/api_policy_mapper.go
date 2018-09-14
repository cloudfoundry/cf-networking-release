package api

import (
	"fmt"
	"policy-server/store"

	"code.cloudfoundry.org/cf-networking-helpers/httperror"
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

func (p *policyMapper) AsStorePolicy(bytes []byte) ([]store.Policy, error) {
	payload := &PoliciesPayload{}
	err := p.Unmarshaler.Unmarshal(bytes, payload)
	if err != nil {
		return []store.Policy{}, fmt.Errorf("unmarshal json: %s", err)
	}

	err = p.PayloadValidator.ValidatePayload(payload)
	if err != nil {
		if metadata, ok := err.(httperror.MetadataError); ok {
			return []store.Policy{}, httperror.NewMetadataError(fmt.Errorf("validate policies: %s", err), metadata.Metadata())
		}
		return []store.Policy{}, fmt.Errorf("validate policies: %s", err)
	}

	var storePolicies []store.Policy
	for _, policy := range payload.Policies {
		storePolicies = append(storePolicies, policy.asStorePolicy())
	}

	return storePolicies, nil
}

func (p *policyMapper) AsBytes(storePolicies []store.Policy) ([]byte, error) {
	// convert store.Policy to api.Policy
	apiPolicies := make([]Policy, len(storePolicies))
	for i, policy := range storePolicies {
		apiPolicies[i] = mapStorePolicy(policy)
	}

	// convert api.Policy payload to bytes
	payload := &PoliciesPayload{
		TotalPolicies:       len(apiPolicies),
		Policies:            apiPolicies,
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
