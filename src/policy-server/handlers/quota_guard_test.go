package handlers_test

import (
	"errors"
	"policy-server/handlers"
	"policy-server/store"
	"policy-server/store/fakes"
	"policy-server/uaa_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("QuotaGuard", func() {
	var (
		quotaGuard       *handlers.QuotaGuard
		fakeStore        *fakes.Store
		policyCollection store.PolicyCollection
		tokenData        uaa_client.CheckTokenResponse
	)
	BeforeEach(func() {
		fakeStore = &fakes.Store{}
		quotaGuard = &handlers.QuotaGuard{
			Store:       fakeStore,
			MaxPolicies: 2,
		}
		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.write"},
			UserID:   "some-developer-guid",
			UserName: "some-developer",
		}
		policyCollection = store.PolicyCollection{
			Policies: []store.Policy{
				{
					Source:      store.Source{ID: "some-app-guid"},
					Destination: store.Destination{ID: "some-other-guid"},
				},
				{
					Source:      store.Source{ID: "some-app-guid"},
					Destination: store.Destination{ID: "yet-another-guid"},
				},
				{
					Source:      store.Source{ID: "some-other-app-guid"},
					Destination: store.Destination{ID: "yet-another-guid"},
				},
			},
		}
		fakeStore.ByGuidsReturns([]store.Policy{}, nil)
	})
	Context("when the user is not an admin", func() {
		Context("when the user is attempting to create an egress policy", func() {
			BeforeEach(func() {
				policyCollection.EgressPolicies = []store.EgressPolicy{
					{
						Source:      store.EgressSource{ID: "some-other-app-guid"},
						Destination: store.EgressDestination{IPRanges: []store.IPRange{{Start: "1.2.3.4", End: "1.2.3.5"}}},
					},
				}
			})

			It("denies policy creation", func() {
				authorized, err := quotaGuard.CheckAccess(policyCollection, tokenData)
				Expect(err).NotTo(HaveOccurred())

				Expect(authorized).To(BeFalse())
			})
		})

		Context("when the additional policies do not exceed the quota", func() {
			It("allows policy creation", func() {
				authorized, err := quotaGuard.CheckAccess(policyCollection, tokenData)
				Expect(err).NotTo(HaveOccurred())

				Expect(authorized).To(BeTrue())
			})
		})
		Context("when the additional policies exceed the quota", func() {
			BeforeEach(func() {
				fakeStore.ByGuidsReturns([]store.Policy{
					{
						Source:      store.Source{ID: "some-other-app-guid"},
						Destination: store.Destination{ID: "yet-another-guid"},
					},
					{
						Source:      store.Source{ID: "some-other-app-guid"},
						Destination: store.Destination{ID: "yet-another-guid"},
					},
				}, nil)
			})
			It("does not allow policy creation", func() {
				authorized, err := quotaGuard.CheckAccess(policyCollection, tokenData)
				Expect(err).NotTo(HaveOccurred())

				Expect(authorized).To(BeFalse())
			})
		})
		Context("when getting the policies by guid fails", func() {
			BeforeEach(func() {
				fakeStore.ByGuidsReturns([]store.Policy{}, errors.New("banana"))
			})
			It("returns an error", func() {
				_, err := quotaGuard.CheckAccess(policyCollection, tokenData)
				Expect(err).To(MatchError("getting policies: banana"))
			})

		})
	})
	Context("when the user is an admin", func() {
		BeforeEach(func() {
			tokenData = uaa_client.CheckTokenResponse{
				Scope:    []string{"network.admin"},
				UserID:   "some-developer-guid",
				UserName: "some-developer",
			}
			fakeStore.ByGuidsReturns([]store.Policy{
				{
					Source:      store.Source{ID: "some-other-app-guid"},
					Destination: store.Destination{ID: "yet-another-guid"},
				},
				{
					Source:      store.Source{ID: "some-other-app-guid"},
					Destination: store.Destination{ID: "yet-another-guid"},
				},
			}, nil)
		})
		It("allows policy creation beyond the max policies", func() {
			authorized, err := quotaGuard.CheckAccess(policyCollection, tokenData)
			Expect(err).NotTo(HaveOccurred())

			Expect(authorized).To(BeTrue())
		})
	})
})
