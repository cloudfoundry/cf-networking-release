package models_test

import (
	"lib/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicySlice", func() {
	var (
		policy1  models.Policy
		policy2  models.Policy
		policies []models.Policy
	)
	BeforeEach(func() {
		policy1 = models.Policy{
			Source: models.Source{
				ID:  "some-app-guid",
				Tag: "AA",
			},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Port:     1234,
				Protocol: "tcp",
			},
		}
		policy2 = models.Policy{
			Source: models.Source{
				ID:  "some-other-app-guid",
				Tag: "BB",
			},
			Destination: models.Destination{
				ID:       "yet-another-app-guid",
				Port:     4567,
				Protocol: "tcp",
			},
		}
		policies = []models.Policy{policy1, policy2}
	})

	Describe("Len", func() {
		It("returns the length of the underlying slice", func() {
			slice := models.PolicySlice(policies)
			Expect(slice.Len()).To(Equal(2))
		})
	})

	Describe("Less", func() {
		BeforeEach(func() {
			policies = []models.Policy{
				{
					Source: models.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: models.Destination{
						ID:       "a",
						Port:     1234,
						Protocol: "tcp",
					},
				},
				{
					Source: models.Source{
						ID:  "a",
						Tag: "b",
					},
					Destination: models.Destination{
						ID:       "a",
						Port:     1234,
						Protocol: "tcp",
					},
				},
				{
					Source: models.Source{
						ID:  "b",
						Tag: "a",
					},
					Destination: models.Destination{
						ID:       "a",
						Port:     1234,
						Protocol: "tcp",
					},
				},
				{
					Source: models.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: models.Destination{
						ID:       "b",
						Port:     1234,
						Protocol: "tcp",
					},
				},
				{
					Source: models.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: models.Destination{
						ID:       "a",
						Port:     1235,
						Protocol: "tcp",
					},
				},
				{
					Source: models.Source{
						ID:  "a",
						Tag: "a",
					},
					Destination: models.Destination{
						ID:       "a",
						Port:     1234,
						Protocol: "udp",
					},
				},
			}

		})
		It("Returns true if the string representation sorts first", func() {
			slice := models.PolicySlice(policies)
			Expect(slice.Less(0, 1)).To(Equal(!slice.Less(1, 0)))
			Expect(slice.Less(0, 2)).To(Equal(!slice.Less(2, 0)))
			Expect(slice.Less(0, 3)).To(Equal(!slice.Less(3, 0)))
			Expect(slice.Less(0, 4)).To(Equal(!slice.Less(4, 0)))
			Expect(slice.Less(0, 5)).To(Equal(!slice.Less(5, 0)))
		})

	})

	Describe("Swap", func() {
		It("swaps the elements at the given index", func() {
			slice := models.PolicySlice(policies)
			slice.Swap(0, 1)
			Expect(policies).To(Equal([]models.Policy{policy2, policy1}))
		})
	})
})
