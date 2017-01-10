package handlers_test

import (
	"errors"
	"policy-server/fakes"
	"policy-server/handlers"
	"policy-server/models"
	"policy-server/uaa_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyFilter", func() {
	var (
		policyFilter  *handlers.PolicyFilter
		fakeCCClient  *fakes.PolicyFilterCCClient
		fakeUAAClient *fakes.UAAClient
		tokenData     uaa_client.CheckTokenResponse
		policies      []models.Policy
	)

	BeforeEach(func() {
		fakeCCClient = &fakes.PolicyFilterCCClient{}
		fakeUAAClient = &fakes.UAAClient{}
		policyFilter = &handlers.PolicyFilter{
			CCClient:  fakeCCClient,
			UAAClient: fakeUAAClient,
		}
		policies = []models.Policy{
			{
				Source: models.Source{
					ID: "app-guid-1",
				},
				Destination: models.Destination{
					ID: "app-guid-2",
				},
			},
			{
				Source: models.Source{
					ID: "app-guid-3",
				},
				Destination: models.Destination{
					ID: "app-guid-4",
				},
			},
		}
		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.write"},
			UserID:   "some-developer-guid",
			UserName: "some-developer",
		}

		appSpaces := map[string]string{
			"app-guid-1": "space-1",
			"app-guid-2": "space-2",
			"app-guid-3": "space-3",
			"app-guid-4": "space-4",
		}

		userSpaces := map[string]struct{}{
			"space-1": struct{}{},
			"space-2": struct{}{},
			"space-3": struct{}{},
		}

		fakeUAAClient.GetTokenReturns("policy-server-token", nil)
		fakeCCClient.GetAppSpacesReturns(appSpaces, nil)
		fakeCCClient.GetUserSpacesReturns(userSpaces, nil)
	})

	Describe("FilterPolicies", func() {
		It("filters the policies by the spaces the user can access", func() {
			filteredPolicies, err := policyFilter.FilterPolicies(policies, tokenData)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
			Expect(fakeCCClient.GetAppSpacesCallCount()).To(Equal(1))

			token, appGUIDs := fakeCCClient.GetAppSpacesArgsForCall(0)
			Expect(token).To(Equal("policy-server-token"))
			Expect(appGUIDs).To(ConsistOf([]string{"app-guid-1", "app-guid-2", "app-guid-3", "app-guid-4"}))

			Expect(fakeCCClient.GetUserSpacesCallCount()).To(Equal(1))

			token, userGUID := fakeCCClient.GetUserSpacesArgsForCall(0)
			Expect(token).To(Equal("policy-server-token"))
			Expect(userGUID).To(Equal("some-developer-guid"))

			expected := []models.Policy{
				{
					Source: models.Source{
						ID: "app-guid-1",
					},
					Destination: models.Destination{
						ID: "app-guid-2",
					},
				},
			}
			Expect(filteredPolicies).To(Equal(expected))
		})

		Context("when the token has network.admin scope", func() {
			BeforeEach(func() {
				tokenData = uaa_client.CheckTokenResponse{
					Scope: []string{"network.admin"},
				}
			})
			It("returns all policies without making extra calls to UAA or CC", func() {
				filtered, err := policyFilter.FilterPolicies(policies, tokenData)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(0))
				Expect(fakeCCClient.GetAppSpacesCallCount()).To(Equal(0))
				Expect(fakeCCClient.GetUserSpacesCallCount()).To(Equal(0))
				Expect(filtered).To(Equal(policies))
			})
		})

		Context("when the getting the app spaces fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetAppSpacesReturns(nil, errors.New("banana"))
			})
			It("returns a useful error", func() {
				filtered, err := policyFilter.FilterPolicies(policies, tokenData)
				Expect(err).To(MatchError("getting app spaces: banana"))
				Expect(filtered).To(BeNil())
			})
		})

		Context("when the getting the user spaces fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetUserSpacesReturns(nil, errors.New("banana"))
			})
			It("returns a useful error", func() {
				filtered, err := policyFilter.FilterPolicies(policies, tokenData)
				Expect(err).To(MatchError("getting user spaces: banana"))
				Expect(filtered).To(BeNil())
			})
		})

		Context("when the getting the policy server token fails", func() {
			BeforeEach(func() {
				fakeUAAClient.GetTokenReturns("", errors.New("banana"))
			})
			It("returns a useful error", func() {
				filtered, err := policyFilter.FilterPolicies(policies, tokenData)
				Expect(err).To(MatchError("getting token: banana"))
				Expect(filtered).To(BeNil())
			})
		})
	})
})
