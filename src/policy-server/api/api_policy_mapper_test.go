package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"policy-server/api"
	"policy-server/store"
	"github.com/onsi/ginkgo/extensions/table"
)

var _ = Describe("ApiPolicyMapper", func() {
	Describe("MapAPIPolicy", func() {
		table.DescribeTable("should map api policy to store policy", func(input api.Policy, expected store.Policy) {
			resultPolicy := api.MapAPIPolicy(input)
			Expect(resultPolicy).To(Equal(expected))
		},
			table.Entry("direct translation",
				api.Policy{
					Source: api.Source{
						ID:  "some-src-id",
						Tag: "some-src-tag",
					},
					Destination: api.Destination{
						ID:       "some-dst-id",
						Tag:      "some-dst-tag",
						Protocol: "some-protocol",
						Ports: api.Ports{
							Start: 8080,
							End:   9000,
						},
					},
				},
				store.Policy{
					Source: store.Source{
						ID:  "some-src-id",
						Tag: "some-src-tag",
					},
					Destination: store.Destination{
						ID:       "some-dst-id",
						Tag:      "some-dst-tag",
						Protocol: "some-protocol",
						Ports: store.Ports{
							Start: 8080,
							End:   9000,
						},
						Port: 0,
					},
				},
			),
			table.Entry("another translation",
				api.Policy{
					Source: api.Source{
						ID:  "some-other-src-id",
						Tag: "some-other-src-tag",
					},
					Destination: api.Destination{
						ID:       "some-other-dst-id",
						Tag:      "some-other-dst-tag",
						Protocol: "some-other-protocol",
						Ports: api.Ports{
							Start: 8088,
							End:   9000,
						},
					},
				},
				store.Policy{
					Source: store.Source{
						ID:  "some-other-src-id",
						Tag: "some-other-src-tag",
					},
					Destination: store.Destination{
						ID:       "some-other-dst-id",
						Tag:      "some-other-dst-tag",
						Protocol: "some-other-protocol",
						Ports: store.Ports{
							Start: 8088,
							End:   9000,
						},
						Port: 0,
					},
				},
			),
		)
	})

	Describe("MapAPIPolicies", func() {
		table.DescribeTable("should map api policies to store policies", func(input []api.Policy, expected []store.Policy) {
			result := api.MapAPIPolicies(input)
			Expect(result).To(Equal(expected))
		},
			table.Entry("direct translation",
				[]api.Policy{{
					Source: api.Source{
						ID:  "some-src-id",
						Tag: "some-src-tag",
					},
					Destination: api.Destination{
						ID:       "some-dst-id",
						Tag:      "some-dst-tag",
						Protocol: "some-protocol",
						Ports: api.Ports{
							Start: 8080,
							End:   9000,
						},
					},
				}},
				[]store.Policy{{
					Source: store.Source{
						ID:  "some-src-id",
						Tag: "some-src-tag",
					},
					Destination: store.Destination{
						ID:       "some-dst-id",
						Tag:      "some-dst-tag",
						Protocol: "some-protocol",
						Ports: store.Ports{
							Start: 8080,
							End:   9000,
						},
						Port: 0,
					},
				}},
			),
			table.Entry("multiple policies",
				[]api.Policy{

					{
						Source: api.Source{
							ID:  "some-src-id",
							Tag: "some-src-tag",
						},
						Destination: api.Destination{
							ID:       "some-dst-id",
							Tag:      "some-dst-tag",
							Protocol: "some-protocol",
							Ports: api.Ports{
								Start: 8080,
								End:   9000,
							},
						},
					},
					{
						Source: api.Source{
							ID:  "some-other-src-id",
							Tag: "some-other-src-tag",
						},
						Destination: api.Destination{
							ID:       "some-other-dst-id",
							Tag:      "some-other-dst-tag",
							Protocol: "some-other-protocol",
							Ports: api.Ports{
								Start: 8088,
								End:   9000,
							},
						},
					},
				},
				[]store.Policy{
					{
						Source: store.Source{
							ID:  "some-src-id",
							Tag: "some-src-tag",
						},
						Destination: store.Destination{
							ID:       "some-dst-id",
							Tag:      "some-dst-tag",
							Protocol: "some-protocol",
							Ports: store.Ports{
								Start: 8080,
								End:   9000,
							},
							Port: 0,
						},
					},
					{
						Source: store.Source{
							ID:  "some-other-src-id",
							Tag: "some-other-src-tag",
						},
						Destination: store.Destination{
							ID:       "some-other-dst-id",
							Tag:      "some-other-dst-tag",
							Protocol: "some-other-protocol",
							Ports: store.Ports{
								Start: 8088,
								End:   9000,
							},
							Port: 0,
						},
					},
				},
			),
		)
	})

	Describe("MapStorePolicy", func() {
		table.DescribeTable("should map store policy to api policy", func(input store.Policy, expected api.Policy) {
			resultPolicy := api.MapStorePolicy(input)
			Expect(resultPolicy).To(Equal(expected))
		},
			table.Entry("direct translation",
				store.Policy{
					Source: store.Source{
						ID:  "some-src-id",
						Tag: "some-src-tag",
					},
					Destination: store.Destination{
						ID:       "some-dst-id",
						Tag:      "some-dst-tag",
						Protocol: "some-protocol",
						Ports: store.Ports{
							Start: 8080,
							End:   9000,
						},
						Port: 0,
					},
				},
				api.Policy{
					Source: api.Source{
						ID:  "some-src-id",
						Tag: "some-src-tag",
					},
					Destination: api.Destination{
						ID:       "some-dst-id",
						Tag:      "some-dst-tag",
						Protocol: "some-protocol",
						Ports: api.Ports{
							Start: 8080,
							End:   9000,
						},
					},
				},
			),
			table.Entry("another translation",
				store.Policy{
					Source: store.Source{
						ID:  "some-other-src-id",
						Tag: "some-other-src-tag",
					},
					Destination: store.Destination{
						ID:       "some-other-dst-id",
						Tag:      "some-other-dst-tag",
						Protocol: "some-other-protocol",
						Ports: store.Ports{
							Start: 8088,
							End:   9000,
						},
						Port: 0,
					},
				},
				api.Policy{
					Source: api.Source{
						ID:  "some-other-src-id",
						Tag: "some-other-src-tag",
					},
					Destination: api.Destination{
						ID:       "some-other-dst-id",
						Tag:      "some-other-dst-tag",
						Protocol: "some-other-protocol",
						Ports: api.Ports{
							Start: 8088,
							End:   9000,
						},
					},
				},
			),
		)
	})

	Describe("MapStorePolicies", func() {
		table.DescribeTable("should map store policies to api policies", func(input []store.Policy, expected []api.Policy) {
			result := api.MapStorePolicies(input)
			Expect(result).To(Equal(expected))
		},
			table.Entry("direct translation",
				[]store.Policy{{
					Source: store.Source{
						ID:  "some-src-id",
						Tag: "some-src-tag",
					},
					Destination: store.Destination{
						ID:       "some-dst-id",
						Tag:      "some-dst-tag",
						Protocol: "some-protocol",
						Ports: store.Ports{
							Start: 8080,
							End:   9000,
						},
						Port: 0,
					},
				}},
				[]api.Policy{{
					Source: api.Source{
						ID:  "some-src-id",
						Tag: "some-src-tag",
					},
					Destination: api.Destination{
						ID:       "some-dst-id",
						Tag:      "some-dst-tag",
						Protocol: "some-protocol",
						Ports: api.Ports{
							Start: 8080,
							End:   9000,
						},
					},
				}},
			),
			table.Entry("multiple policies",
				[]store.Policy{{
					Source: store.Source{
						ID:  "some-src-id",
						Tag: "some-src-tag",
					},
					Destination: store.Destination{
						ID:       "some-dst-id",
						Tag:      "some-dst-tag",
						Protocol: "some-protocol",
						Ports: store.Ports{
							Start: 8080,
							End:   9000,
						},
						Port: 0,
					},
				}, {
					Source: store.Source{
						ID:  "some-other-src-id",
						Tag: "some-other-src-tag",
					},
					Destination: store.Destination{
						ID:       "some-other-dst-id",
						Tag:      "some-other-dst-tag",
						Protocol: "some-other-protocol",
						Ports: store.Ports{
							Start: 8088,
							End:   9000,
						},
						Port: 0,
					},
				},
				},
				[]api.Policy{{
					Source: api.Source{
						ID:  "some-src-id",
						Tag: "some-src-tag",
					},
					Destination: api.Destination{
						ID:       "some-dst-id",
						Tag:      "some-dst-tag",
						Protocol: "some-protocol",
						Ports: api.Ports{
							Start: 8080,
							End:   9000,
						},
					},
				}, {
					Source: api.Source{
						ID:  "some-other-src-id",
						Tag: "some-other-src-tag",
					},
					Destination: api.Destination{
						ID:       "some-other-dst-id",
						Tag:      "some-other-dst-tag",
						Protocol: "some-other-protocol",
						Ports: api.Ports{
							Start: 8088,
							End:   9000,
						},
					},
				},
				},
			),
		)
	})

	Describe("MapStoreTag", func() {
		table.DescribeTable("should map store tags to api tags", func(input store.Tag, expected api.Tag) {
			result := api.MapStoreTag(input)
			Expect(result).To(Equal(expected))
		},
			table.Entry("direct translation",
				store.Tag{
					ID:  "some-id",
					Tag: "some-tag",
				},
				api.Tag{
					ID:  "some-id",
					Tag: "some-tag",
				},
			),
			table.Entry("direct translation",
				store.Tag{
					ID:  "some-other-id",
					Tag: "some-other-tag",
				},
				api.Tag{
					ID:  "some-other-id",
					Tag: "some-other-tag",
				},
			),
		)
	})

	Describe("MapStoreTags", func() {
		table.DescribeTable("should map store tags to api tags", func(input []store.Tag, expected []api.Tag) {
			result := api.MapStoreTags(input)
			Expect(result).To(Equal(expected))
		},
			table.Entry("direct translation",
				[]store.Tag{{
					ID:  "some-id",
					Tag: "some-tag",
				}},
				[]api.Tag{{
					ID:  "some-id",
					Tag: "some-tag",
				}},
			),
			table.Entry("direct translation",
				[]store.Tag{{
					ID:  "some-id",
					Tag: "some-tag",
				},
					{
						ID:  "some-other-id",
						Tag: "some-other-tag",
					}},
				[]api.Tag{{
					ID:  "some-id",
					Tag: "some-tag",
				},
					{
					ID:  "some-other-id",
					Tag: "some-other-tag",
				}},
			),
		)
	})

})
