package api_test

import (
	"policy-server/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicySlice", func() {
	var (
		policy1  api.Policy
		policy2  api.Policy
		policies []api.Policy
	)
	BeforeEach(func() {
		policy1 = api.Policy{
			Source: api.Source{
				ID:  "some-app-guid",
				Tag: "AA",
			},
			Destination: api.Destination{
				ID: "some-other-app-guid",
				Ports: api.Ports{
					Start: 1234,
					End:   1234,
				},
				Protocol: "tcp",
			},
		}
		policy2 = api.Policy{
			Source: api.Source{
				ID:  "some-other-app-guid",
				Tag: "BB",
			},
			Destination: api.Destination{
				ID: "yet-another-app-guid",
				Ports: api.Ports{
					Start: 4567,
					End:   4567,
				},
				Protocol: "tcp",
			},
		}
		policies = []api.Policy{policy1, policy2}
	})

	Describe("Len", func() {
		It("returns the length of the underlying slice", func() {
			slice := api.PolicySlice(policies)
			Expect(slice.Len()).To(Equal(2))
		})
	})

	Describe("Less", func() {
		BeforeEach(func() {
			policies = []api.Policy{
				{
					Source: api.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: api.Destination{
						ID: "a",
						Ports: api.Ports{
							Start: 1234,
							End:   1234,
						},
						Protocol: "tcp",
					},
				},
				{
					Source: api.Source{
						ID:  "a",
						Tag: "b",
					},
					Destination: api.Destination{
						ID: "a",
						Ports: api.Ports{
							Start: 4321,
							End:   4321,
						},
						Protocol: "tcp",
					},
				},
				{
					Source: api.Source{
						ID:  "b",
						Tag: "a",
					},
					Destination: api.Destination{
						ID: "a",
						Ports: api.Ports{
							Start: 1234,
							End:   1234,
						},
						Protocol: "tcp",
					},
				},
				{
					Source: api.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: api.Destination{
						ID: "b",
						Ports: api.Ports{
							Start: 1234,
							End:   1234,
						},
						Protocol: "tcp",
					},
				},
				{
					Source: api.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: api.Destination{
						ID: "a",
						Ports: api.Ports{
							Start: 1235,
							End:   1235,
						},
						Protocol: "tcp",
					},
				},
				{
					Source: api.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: api.Destination{
						ID: "a",
						Ports: api.Ports{
							Start: 1234,
							End:   1234,
						},
						Protocol: "udp",
					},
				},
			}

		})
		It("Returns true if the string representation sorts first", func() {
			slice := api.PolicySlice(policies)
			Expect(slice.Less(0, 1)).To(Equal(!slice.Less(1, 0)))
			Expect(slice.Less(0, 2)).To(Equal(!slice.Less(2, 0)))
			Expect(slice.Less(0, 3)).To(Equal(!slice.Less(3, 0)))
			Expect(slice.Less(0, 4)).To(Equal(!slice.Less(4, 0)))
			Expect(slice.Less(0, 5)).To(Equal(!slice.Less(5, 0)))
		})

	})

	Describe("Swap", func() {
		It("swaps the elements at the given index", func() {
			slice := api.PolicySlice(policies)
			slice.Swap(0, 1)
			Expect(policies).To(Equal([]api.Policy{policy2, policy1}))
		})
	})
})
