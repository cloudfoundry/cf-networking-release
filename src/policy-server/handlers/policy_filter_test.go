package handlers_test

import (
	"errors"
	"policy-server/handlers"
	"policy-server/handlers/fakes"
	"policy-server/api"
	"policy-server/uaa_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyFilter", func() {
	var (
		policyFilter  *handlers.PolicyFilter
		fakeCCClient  *fakes.CCClient
		fakeUAAClient *fakes.UAAClient
		tokenData     uaa_client.CheckTokenResponse
		policies      []api.Policy
	)

	BeforeEach(func() {
		fakeCCClient = &fakes.CCClient{}
		fakeUAAClient = &fakes.UAAClient{}
		policyFilter = &handlers.PolicyFilter{
			CCClient:  fakeCCClient,
			UAAClient: fakeUAAClient,
			ChunkSize: 100,
		}
		policies = []api.Policy{
			{
				Source: api.Source{
					ID: "app-guid-1",
				},
				Destination: api.Destination{
					ID: "app-guid-2",
				},
			},
			{
				Source: api.Source{
					ID: "app-guid-3",
				},
				Destination: api.Destination{
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

			expected := []api.Policy{
				{
					Source: api.Source{
						ID: "app-guid-1",
					},
					Destination: api.Destination{
						ID: "app-guid-2",
					},
				},
			}
			Expect(filteredPolicies).To(Equal(expected))
		})

		Context("when the filter results in zero policies", func() {
			BeforeEach(func() {
				fakeCCClient.GetUserSpacesReturns(map[string]struct{}{}, nil)
			})

			It("returns a non-null, but empty, slice of policies", func() {
				filteredPolicies, err := policyFilter.FilterPolicies(policies, tokenData)
				Expect(err).NotTo(HaveOccurred())
				Expect(filteredPolicies).NotTo(BeNil())
				Expect(filteredPolicies).To(HaveLen(0))
			})
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

		Context("when the number of unique app guids is greater than the chunk size", func() {
			BeforeEach(func() {
				policyFilter.ChunkSize = 1
			})
			It("chunks the guids and makes multiple requests to CC", func() {
				filtered, err := policyFilter.FilterPolicies(policies, tokenData)
				Expect(err).NotTo(HaveOccurred())

				expected := []api.Policy{
					{
						Source: api.Source{
							ID: "app-guid-1",
						},
						Destination: api.Destination{
							ID: "app-guid-2",
						},
					},
				}

				Expect(fakeCCClient.GetAppSpacesCallCount()).To(Equal(4))

				appGUIDs := []string{}
				for i := 0; i < 4; i++ {
					_, guids := fakeCCClient.GetAppSpacesArgsForCall(i)
					appGUIDs = append(appGUIDs, guids...)
				}
				Expect(appGUIDs).To(ConsistOf("app-guid-1",
					"app-guid-2",
					"app-guid-3",
					"app-guid-4",
				))

				Expect(filtered).To(Equal(expected))
			})
		})
	})
})
