package api_v0_internal_test

import (
	"code.cloudfoundry.org/policy-server/api/api_v0_internal"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicySlice", func() {
	var (
		policy1  api_v0_internal.Policy
		policy2  api_v0_internal.Policy
		policies []api_v0_internal.Policy
	)
	BeforeEach(func() {
		policy1 = api_v0_internal.Policy{
			Source: api_v0_internal.Source{
				ID:  "some-app-guid",
				Tag: "AA",
			},
			Destination: api_v0_internal.Destination{
				ID: "some-other-app-guid",
				Ports: api_v0_internal.Ports{
					Start: 1234,
					End:   1234,
				},
				Protocol: "tcp",
			},
		}
		policy2 = api_v0_internal.Policy{
			Source: api_v0_internal.Source{
				ID:  "some-other-app-guid",
				Tag: "BB",
			},
			Destination: api_v0_internal.Destination{
				ID: "yet-another-app-guid",
				Ports: api_v0_internal.Ports{
					Start: 4567,
					End:   4567,
				},
				Protocol: "tcp",
			},
		}
		policies = []api_v0_internal.Policy{policy1, policy2}
	})

	Describe("Len", func() {
		It("returns the length of the underlying slice", func() {
			slice := api_v0_internal.PolicySlice(policies)
			Expect(slice.Len()).To(Equal(2))
		})
	})

	Describe("Less", func() {
		BeforeEach(func() {
			policies = []api_v0_internal.Policy{
				{
					Source: api_v0_internal.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: api_v0_internal.Destination{
						ID: "a",
						Ports: api_v0_internal.Ports{
							Start: 1234,
							End:   1234,
						},
						Protocol: "tcp",
					},
				},
				{
					Source: api_v0_internal.Source{
						ID:  "a",
						Tag: "b",
					},
					Destination: api_v0_internal.Destination{
						ID: "a",
						Ports: api_v0_internal.Ports{
							Start: 4321,
							End:   4321,
						},
						Protocol: "tcp",
					},
				},
				{
					Source: api_v0_internal.Source{
						ID:  "b",
						Tag: "a",
					},
					Destination: api_v0_internal.Destination{
						ID: "a",
						Ports: api_v0_internal.Ports{
							Start: 1234,
							End:   1234,
						},
						Protocol: "tcp",
					},
				},
				{
					Source: api_v0_internal.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: api_v0_internal.Destination{
						ID: "b",
						Ports: api_v0_internal.Ports{
							Start: 1234,
							End:   1234,
						},
						Protocol: "tcp",
					},
				},
				{
					Source: api_v0_internal.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: api_v0_internal.Destination{
						ID: "a",
						Ports: api_v0_internal.Ports{
							Start: 1235,
							End:   1235,
						},
						Protocol: "tcp",
					},
				},
				{
					Source: api_v0_internal.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: api_v0_internal.Destination{
						ID: "a",
						Ports: api_v0_internal.Ports{
							Start: 1234,
							End:   1234,
						},
						Protocol: "udp",
					},
				},
			}

		})
		It("Returns true if the string representation sorts first", func() {
			slice := api_v0_internal.PolicySlice(policies)
			Expect(slice.Less(0, 1)).To(Equal(!slice.Less(1, 0)))
			Expect(slice.Less(0, 2)).To(Equal(!slice.Less(2, 0)))
			Expect(slice.Less(0, 3)).To(Equal(!slice.Less(3, 0)))
			Expect(slice.Less(0, 4)).To(Equal(!slice.Less(4, 0)))
			Expect(slice.Less(0, 5)).To(Equal(!slice.Less(5, 0)))
		})

	})

	Describe("Swap", func() {
		It("swaps the elements at the given index", func() {
			slice := api_v0_internal.PolicySlice(policies)
			slice.Swap(0, 1)
			Expect(policies).To(Equal([]api_v0_internal.Policy{policy2, policy1}))
		})
	})
})
