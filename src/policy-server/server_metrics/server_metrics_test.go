package server_metrics_test

import (
	"policy-server/fakes"
	"policy-server/models"
	"policy-server/server_metrics"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewTotalPoliciesSource", func() {
	var (
		allPolicies   []models.Policy
		fakeDataStore *fakes.Store
	)

	BeforeEach(func() {
		allPolicies = []models.Policy{{
			Source: models.Source{ID: "some-app-guid", Tag: "some-tag"},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Tag:      "some-other-tag",
				Protocol: "tcp",
				Port:     8080,
			},
		}, {
			Source: models.Source{ID: "another-app-guid"},
			Destination: models.Destination{
				ID:       "some-other-app-guid",
				Protocol: "udp",
				Port:     1234,
			},
		}}

		fakeDataStore = &fakes.Store{}
		fakeDataStore.AllReturns(allPolicies, nil)
	})

	Describe("Getter", func() {
		It("returns the total number of policies in the datastore", func() {
			source := server_metrics.NewTotalPoliciesSource(fakeDataStore)
			Expect(source.Name).To(Equal("totalPolicies"))
			Expect(source.Unit).To(Equal(""))

			value, err := source.Getter()
			Expect(err).NotTo(HaveOccurred())

			Expect(value).To(Equal(2.0))
		})
	})
})
