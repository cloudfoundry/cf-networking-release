package server_metrics_test

import (
	"policy-server/server_metrics"
	"policy-server/server_metrics/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"policy-server/store"
)

var _ = Describe("NewTotalPoliciesSource", func() {
	var (
		allPolicies   []store.Policy
		fakeDataStore *fakes.ListStore
	)

	BeforeEach(func() {
		allPolicies = []store.Policy{{
			Source: store.Source{ID: "some-app-guid", Tag: "some-tag"},
			Destination: store.Destination{
				ID:       "some-other-app-guid",
				Tag:      "some-other-tag",
				Protocol: "tcp",
				Ports: store.Ports{
					Start: 8080,
					End:   8080,
				},
			},
		}, {
			Source: store.Source{ID: "another-app-guid"},
			Destination: store.Destination{
				ID:       "some-other-app-guid",
				Protocol: "udp",
				Ports: store.Ports{
					Start: 1234,
					End:   1234,
				},
			},
		}}

		fakeDataStore = &fakes.ListStore{}
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
