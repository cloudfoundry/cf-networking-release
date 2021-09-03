package handlers_test

import (
	"errors"

	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/store/fakes"
	"code.cloudfoundry.org/policy-server/uaa_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("QuotaGuard", func() {
	var (
		quotaGuard *handlers.QuotaGuard
		fakeStore  *fakes.Store
		policies   []store.Policy
		tokenData  uaa_client.CheckTokenResponse
	)
	BeforeEach(func() {
		fakeStore = &fakes.Store{}
		quotaGuard = &handlers.QuotaGuard{
			Store:       fakeStore,
			MaxPolicies: 2,
		}
		tokenData = uaa_client.CheckTokenResponse{
			Scope: []string{"network.write"},
		}
		policies = []store.Policy{
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
		}
		fakeStore.ByGuidsReturns([]store.Policy{}, nil)
	})
	Context("when the subject is not an admin", func() {
		Context("when the additional policies do not exceed the quota", func() {
			It("allows policy creation", func() {
				authorized, err := quotaGuard.CheckAccess(policies, tokenData)
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
				authorized, err := quotaGuard.CheckAccess(policies, tokenData)
				Expect(err).NotTo(HaveOccurred())

				Expect(authorized).To(BeFalse())
			})
		})
		Context("when getting the policies by guid fails", func() {
			BeforeEach(func() {
				fakeStore.ByGuidsReturns([]store.Policy{}, errors.New("banana"))
			})
			It("returns an error", func() {
				_, err := quotaGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting policies: banana"))
			})

		})
	})
	Context("when the subject is an admin", func() {
		BeforeEach(func() {
			tokenData = uaa_client.CheckTokenResponse{
				Scope: []string{"network.admin"},
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
			authorized, err := quotaGuard.CheckAccess(policies, tokenData)
			Expect(err).NotTo(HaveOccurred())

			Expect(authorized).To(BeTrue())
		})
	})
})
