package handlers_test

import (
	"errors"

	ccfakes "code.cloudfoundry.org/policy-server/cc_client/fakes"
	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/uaa_client"
	uaafakes "code.cloudfoundry.org/policy-server/uaa_client/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyFilter", func() {
	var (
		policyFilter  *handlers.PolicyFilter
		fakeCCClient  *ccfakes.CCClient
		fakeUAAClient *uaafakes.UAAClient
		tokenData     uaa_client.CheckTokenResponse
		policies      []store.Policy
	)

	BeforeEach(func() {
		fakeCCClient = &ccfakes.CCClient{}
		fakeUAAClient = &uaafakes.UAAClient{}
		policyFilter = &handlers.PolicyFilter{
			CCClient:  fakeCCClient,
			UAAClient: fakeUAAClient,
			ChunkSize: 100,
		}
		policies = []store.Policy{
			{
				Source: store.Source{
					ID: "app-guid-1",
				},
				Destination: store.Destination{
					ID: "app-guid-2",
				},
			},
			{
				Source: store.Source{
					ID: "app-guid-3",
				},
				Destination: store.Destination{
					ID: "app-guid-4",
				},
			},
		}
		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.write"},
			Subject:  "some-developer-guid",
			UserName: "some-developer",
		}

		appSpaces := map[string]string{
			"app-guid-1": "space-1",
			"app-guid-2": "space-2",
			"app-guid-3": "space-3",
			"app-guid-4": "space-4",
		}

		subjectSpaces := map[string]struct{}{
			"space-1": {},
			"space-2": {},
			"space-3": {},
		}

		fakeUAAClient.GetTokenReturns("policy-server-token", nil)
		fakeCCClient.GetAppSpacesReturns(appSpaces, nil)
		fakeCCClient.GetSubjectSpacesReturns(subjectSpaces, nil)
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

			Expect(fakeCCClient.GetSubjectSpacesCallCount()).To(Equal(1))

			token, subjectId := fakeCCClient.GetSubjectSpacesArgsForCall(0)
			Expect(token).To(Equal("policy-server-token"))
			Expect(subjectId).To(Equal("some-developer-guid"))

			expected := []store.Policy{
				{
					Source: store.Source{
						ID: "app-guid-1",
					},
					Destination: store.Destination{
						ID: "app-guid-2",
					},
				},
			}
			Expect(filteredPolicies).To(Equal(expected))
		})

		Context("when the token has a client as the subject", func() {
			BeforeEach(func() {
				tokenData = uaa_client.CheckTokenResponse{
					ClientID: "some-client-id",
					Subject:  "some-client-id",
				}
			})

			It("filters the policies by the spaces the client can access", func() {
				filteredPolicies, err := policyFilter.FilterPolicies(policies, tokenData)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
				Expect(fakeCCClient.GetAppSpacesCallCount()).To(Equal(1))

				token, appGUIDs := fakeCCClient.GetAppSpacesArgsForCall(0)
				Expect(token).To(Equal("policy-server-token"))
				Expect(appGUIDs).To(ConsistOf([]string{"app-guid-1", "app-guid-2", "app-guid-3", "app-guid-4"}))

				Expect(fakeCCClient.GetSubjectSpacesCallCount()).To(Equal(1))

				token, subjectId := fakeCCClient.GetSubjectSpacesArgsForCall(0)
				Expect(token).To(Equal("policy-server-token"))
				Expect(subjectId).To(Equal("some-client-id"))

				expected := []store.Policy{
					{
						Source: store.Source{
							ID: "app-guid-1",
						},
						Destination: store.Destination{
							ID: "app-guid-2",
						},
					},
				}
				Expect(filteredPolicies).To(Equal(expected))
			})
		})

		Context("when the filter results in zero policies", func() {
			BeforeEach(func() {
				fakeCCClient.GetSubjectSpacesReturns(map[string]struct{}{}, nil)
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
				Expect(fakeCCClient.GetSubjectSpacesCallCount()).To(Equal(0))
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

		Context("when the getting the subject spaces fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetSubjectSpacesReturns(nil, errors.New("banana"))
			})
			It("returns a useful error", func() {
				filtered, err := policyFilter.FilterPolicies(policies, tokenData)
				Expect(err).To(MatchError("getting subject spaces: banana"))
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

				expected := []store.Policy{
					{
						Source: store.Source{
							ID: "app-guid-1",
						},
						Destination: store.Destination{
							ID: "app-guid-2",
						},
					},
				}

				Expect(fakeCCClient.GetAppSpacesCallCount()).To(Equal(4))

				var appGUIDs []string
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
