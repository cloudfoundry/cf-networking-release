package handlers_test

import (
	"errors"
	"lib/models"
	"policy-server/fakes"
	"policy-server/handlers"
	"policy-server/uaa_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyGuard", func() {
	var (
		policyGuard   *handlers.PolicyGuard
		fakeCCClient  *fakes.PolicyGuardCCClient
		fakeUAAClient *fakes.UAAClient
		tokenData     uaa_client.CheckTokenResponse
		policies      []models.Policy
		spaceGuids    []string
		spaces        []models.Space
	)

	BeforeEach(func() {
		fakeCCClient = &fakes.PolicyGuardCCClient{}
		fakeUAAClient = &fakes.UAAClient{}
		policyGuard = &handlers.PolicyGuard{
			CCClient:  fakeCCClient,
			UAAClient: fakeUAAClient,
		}
		policies = []models.Policy{
			{
				Source: models.Source{
					ID: "some-app-guid",
				},
				Destination: models.Destination{
					ID: "some-other-guid",
				},
			},
			{
				Source: models.Source{
					ID: "some-app-guid",
				},
				Destination: models.Destination{
					ID: "yet-another-guid",
				},
			},
		}
		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.write"},
			UserID:   "some-developer-guid",
			UserName: "some-developer",
		}
		spaceGuids = []string{"space-guid-1", "space-guid-2", "space-guid-3"}
		spaces = []models.Space{
			{
				Name:    "space-1",
				OrgGuid: "org-guid-1",
			},
			{
				Name:    "space-2",
				OrgGuid: "org-guid-2",
			},
			{
				Name:    "space-3",
				OrgGuid: "org-guid-3",
			},
		}

		fakeUAAClient.GetTokenReturns("policy-server-token", nil)
		fakeCCClient.GetSpaceGuidsReturns(spaceGuids, nil)
		fakeCCClient.GetSpacesReturns(spaces, nil)
		fakeCCClient.GetUserSpacesReturns(spaces, nil)
	})

	Describe("CheckAccess", func() {

		It("checks that the user can access all apps references in policies", func() {
			authorized, err := policyGuard.CheckAccess(policies, tokenData)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
			Expect(fakeCCClient.GetSpaceGuidsCallCount()).To(Equal(1))
			token, appGuids := fakeCCClient.GetSpaceGuidsArgsForCall(0)
			Expect(token).To(Equal("policy-server-token"))
			Expect(appGuids).To(ConsistOf([]string{"some-app-guid", "some-other-guid", "yet-another-guid"}))
			Expect(fakeCCClient.GetSpacesCallCount()).To(Equal(1))
			token, actualSpaceGuids := fakeCCClient.GetSpacesArgsForCall(0)
			Expect(token).To(Equal("policy-server-token"))
			Expect(actualSpaceGuids).To(Equal(spaceGuids))
			Expect(fakeCCClient.GetUserSpacesCallCount()).To(Equal(1))
			token, userGuid, checkUserSpaces := fakeCCClient.GetUserSpacesArgsForCall(0)
			Expect(token).To(Equal("policy-server-token"))
			Expect(userGuid).To(Equal("some-developer-guid"))
			Expect(checkUserSpaces).To(Equal(spaces))
			Expect(authorized).To(BeTrue())
		})

		Context("when the token has network.admin scope", func() {
			BeforeEach(func() {
				tokenData = uaa_client.CheckTokenResponse{
					Scope: []string{"network.admin"},
				}
			})
			It("returns successfully without making extra calls to UAA or CC", func() {
				authorized, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(0))
				Expect(fakeCCClient.GetSpacesCallCount()).To(Equal(0))
				Expect(fakeCCClient.GetUserSpacesCallCount()).To(Equal(0))
				Expect(authorized).To(BeTrue())
			})
		})

		Context("when the user cannot access one or more apps", func() {
			BeforeEach(func() {
				spaces = []models.Space{
					{
						Name:    "space-1",
						OrgGuid: "org-guid-1",
					},
					{
						Name:    "space-2",
						OrgGuid: "org-guid-2",
					},
				}
				fakeCCClient.GetUserSpacesReturns(spaces, nil)
			})
			It("returns false", func() {
				authorized, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).NotTo(HaveOccurred())
				Expect(authorized).To(BeFalse())
			})
		})

		Context("when the getting the policy server token fails", func() {
			BeforeEach(func() {
				fakeUAAClient.GetTokenReturns("", errors.New("banana"))
			})
			It("returns a useful error", func() {
				_, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting token: banana"))
			})
		})

		Context("when the getting the space guids fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetSpaceGuidsReturns(nil, errors.New("banana"))
			})
			It("returns an useful error", func() {
				_, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting space guids: banana"))
			})
		})

		Context("when the getting all the spaces fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetSpacesReturns(nil, errors.New("banana"))
			})
			It("returns an useful error", func() {
				_, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting spaces: banana"))
			})
		})

		Context("when the getting the users spaces fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetUserSpacesReturns(nil, errors.New("banana"))
			})
			It("returns an useful error", func() {
				_, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting user spaces: banana"))
			})
		})
	})
})
