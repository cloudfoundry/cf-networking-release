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
		space1        models.Space
		space2        models.Space
		space3        models.Space
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
		space1 = models.Space{
			Name:    "space-1",
			OrgGuid: "org-guid-1",
		}
		space2 = models.Space{
			Name:    "space-2",
			OrgGuid: "org-guid-2",
		}
		space3 = models.Space{
			Name:    "space-3",
			OrgGuid: "org-guid-3",
		}
		spaces = []models.Space{space1, space2, space3}

		fakeUAAClient.GetTokenReturns("policy-server-token", nil)
		fakeCCClient.GetSpaceGuidsReturns(spaceGuids, nil)
		fakeCCClient.GetSpaceStub = func(token, spaceGuid string) (models.Space, error) {
			switch spaceGuid {
			case "space-guid-1":
				{
					return space1, nil
				}
			case "space-guid-2":
				{
					return space2, nil
				}
			case "space-guid-3":
				{
					return space3, nil
				}
			default:
				{
					return models.Space{}, errors.New("stub called with unexpected guid")
				}
			}
		}
		fakeCCClient.GetUserSpaceStub = func(token, userGuid string, space models.Space) (models.Space, error) {
			switch space {
			case space1:
				{
					return space1, nil
				}
			case space2:
				{
					return space2, nil
				}
			case space3:
				{
					return space3, nil
				}
			default:
				{
					return models.Space{}, errors.New("stub called with unexpected guid")
				}
			}
		}
	})

	Describe("CheckAccess", func() {

		It("checks that the user can access all apps references in policies", func() {
			err := policyGuard.CheckAccess(policies, tokenData)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
			Expect(fakeCCClient.GetSpaceGuidsCallCount()).To(Equal(1))
			token, appGuids := fakeCCClient.GetSpaceGuidsArgsForCall(0)
			Expect(token).To(Equal("policy-server-token"))
			Expect(appGuids).To(ConsistOf([]string{"some-app-guid", "some-other-guid", "yet-another-guid"}))
			Expect(fakeCCClient.GetSpaceCallCount()).To(Equal(3))
			token, guid := fakeCCClient.GetSpaceArgsForCall(0)
			Expect(token).To(Equal("policy-server-token"))
			Expect(guid).To(Equal("space-guid-1"))
			token, guid = fakeCCClient.GetSpaceArgsForCall(1)
			Expect(token).To(Equal("policy-server-token"))
			Expect(guid).To(Equal("space-guid-2"))
			token, guid = fakeCCClient.GetSpaceArgsForCall(2)
			Expect(token).To(Equal("policy-server-token"))
			Expect(guid).To(Equal("space-guid-3"))
			Expect(fakeCCClient.GetUserSpaceCallCount()).To(Equal(3))
			token, userGuid, checkUserSpace := fakeCCClient.GetUserSpaceArgsForCall(0)
			Expect(token).To(Equal("policy-server-token"))
			Expect(userGuid).To(Equal("some-developer-guid"))
			Expect(checkUserSpace).To(Equal(space1))
			token, userGuid, checkUserSpace = fakeCCClient.GetUserSpaceArgsForCall(1)
			Expect(token).To(Equal("policy-server-token"))
			Expect(userGuid).To(Equal("some-developer-guid"))
			Expect(checkUserSpace).To(Equal(space2))
			token, userGuid, checkUserSpace = fakeCCClient.GetUserSpaceArgsForCall(2)
			Expect(token).To(Equal("policy-server-token"))
			Expect(userGuid).To(Equal("some-developer-guid"))
			Expect(checkUserSpace).To(Equal(space3))
		})

		Context("when the token has network.admin scope", func() {
			BeforeEach(func() {
				tokenData = uaa_client.CheckTokenResponse{
					Scope: []string{"network.admin"},
				}
			})
			It("returns successfully without making extra calls to UAA or CC", func() {
				err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(0))
				Expect(fakeCCClient.GetSpaceCallCount()).To(Equal(0))
				Expect(fakeCCClient.GetUserSpaceCallCount()).To(Equal(0))
			})
		})

		Context("when the getting the policy server token fails", func() {
			BeforeEach(func() {
				fakeUAAClient.GetTokenReturns("", errors.New("banana"))
			})
			It("returns a useful error", func() {
				err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting token: banana"))
			})
		})

		Context("when the getting the space guids fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetSpaceGuidsReturns(nil, errors.New("banana"))
			})
			It("returns an useful error", func() {
				err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting space guids: banana"))
			})
		})

		Context("when the getting all the spaces fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetSpaceReturns(models.Space{}, errors.New("banana"))
			})
			It("returns an useful error", func() {
				err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting space with guid space-guid-1: banana"))
			})
		})

		Context("when the getting the users spaces fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetUserSpaceReturns(models.Space{}, errors.New("banana"))
			})
			It("returns an useful error", func() {
				err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting user space space-1 in org org-guid-1: banana"))
			})
		})
	})
})
