package api

import "policy-server/store"

func MapAPIPolicy(policy Policy) store.Policy {
	return store.Policy{
		Source: store.Source{
			ID:  policy.Source.ID,
			Tag: policy.Source.Tag,
		},
		Destination: store.Destination{
			ID:       policy.Destination.ID,
			Tag:      policy.Destination.Tag,
			Protocol: policy.Destination.Protocol,
			Ports: store.Ports{
				Start: policy.Destination.Ports.Start,
				End:   policy.Destination.Ports.End,
			},
		},
	}
}

func MapAPIPolicies(policies []Policy) []store.Policy {
	storePolicies := []store.Policy{}

	for _, policy := range policies {
		storePolicies = append(storePolicies, MapAPIPolicy(policy))
	}

	return storePolicies
}

func MapStorePolicy(storePolicy store.Policy) Policy {
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

func MapStorePolicies(storePolicies []store.Policy) []Policy {
	policies := []Policy{}

	for _, policy := range storePolicies {
		policies = append(policies, MapStorePolicy(policy))
	}

	return policies
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
