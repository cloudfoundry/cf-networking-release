package handlers_test

import (
	"errors"

	"code.cloudfoundry.org/policy-server/cc_client"
	ccfakes "code.cloudfoundry.org/policy-server/cc_client/fakes"
	"code.cloudfoundry.org/policy-server/handlers"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/uaa_client"
	uaafakes "code.cloudfoundry.org/policy-server/uaa_client/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyGuard", func() {
	var (
		policyGuard   *handlers.PolicyGuard
		fakeCCClient  *ccfakes.CCClient
		fakeUAAClient *uaafakes.UAAClient
		tokenData     uaa_client.CheckTokenResponse
		policies      []store.Policy
		spaceGUIDs    []string
		space1        cc_client.SpaceResponse
		space2        cc_client.SpaceResponse
		space3        cc_client.SpaceResponse
	)

	BeforeEach(func() {
		fakeCCClient = &ccfakes.CCClient{}
		fakeUAAClient = &uaafakes.UAAClient{}
		policyGuard = &handlers.PolicyGuard{
			CCClient:  fakeCCClient,
			UAAClient: fakeUAAClient,
		}
		policies = []store.Policy{
			{
				Source: store.Source{
					ID: "some-app-guid",
				},
				Destination: store.Destination{
					ID: "some-other-guid",
				},
			},
			{
				Source: store.Source{
					ID: "some-app-guid",
				},
				Destination: store.Destination{
					ID: "yet-another-guid",
				},
			},
		}
		tokenData = uaa_client.CheckTokenResponse{
			Scope:    []string{"network.write"},
			Subject:  "some-developer-guid",
			UserName: "some-developer",
		}
		spaceGUIDs = []string{"space-guid-1", "space-guid-2", "space-guid-3"}
		space1 = cc_client.SpaceResponse{
			Entity: cc_client.SpaceEntity{
				Name:             "space-1",
				OrganizationGUID: "org-guid-1",
			},
		}
		space2 = cc_client.SpaceResponse{
			Entity: cc_client.SpaceEntity{
				Name:             "space-2",
				OrganizationGUID: "org-guid-2",
			}}
		space3 = cc_client.SpaceResponse{
			Entity: cc_client.SpaceEntity{
				Name:             "space-3",
				OrganizationGUID: "org-guid-3",
			}}

		fakeUAAClient.GetTokenReturns("policy-server-token", nil)
		fakeCCClient.GetSpaceGUIDsReturns(spaceGUIDs, nil)
		fakeCCClient.GetSpaceStub = func(token, spaceGUID string) (*cc_client.SpaceResponse, error) {
			switch spaceGUID {
			case "space-guid-1":
				{
					return &space1, nil
				}
			case "space-guid-2":
				{
					return &space2, nil
				}
			case "space-guid-3":
				{
					return &space3, nil
				}
			default:
				{
					return nil, errors.New("stub called with unexpected guid")
				}
			}
		}
		fakeCCClient.GetSubjectSpaceStub = func(token, subjectId string, space cc_client.SpaceResponse) (*cc_client.SpaceResource, error) {
			switch space {
			case space1:
				{
					return &cc_client.SpaceResource{Entity: space1.Entity}, nil
				}
			case space2:
				{
					return &cc_client.SpaceResource{Entity: space2.Entity}, nil
				}
			case space3:
				{
					return &cc_client.SpaceResource{Entity: space3.Entity}, nil
				}
			default:
				{
					return nil, errors.New("stub called with unexpected guid")
				}
			}
		}
	})

	Describe("IsNetworkAdmin", func() {
		Context("when the subject is network.admin", func() {
			BeforeEach(func() {
				tokenData = uaa_client.CheckTokenResponse{
					Scope: []string{"network.admin"},
				}
			})
			It("returns true", func() {
				authorized := policyGuard.IsNetworkAdmin(tokenData)
				Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(0))
				Expect(authorized).To(BeTrue())
			})
		})

		Context("when the subject is not network.admin", func() {
			BeforeEach(func() {
				tokenData = uaa_client.CheckTokenResponse{
					Scope: []string{"not-network.not-admin"},
				}
			})
			It("returns false", func() {
				authorized := policyGuard.IsNetworkAdmin(tokenData)
				Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(0))
				Expect(authorized).To(BeFalse())
			})
		})

	})

	Describe("CheckAccess", func() {
		It("checks that the user can access all apps references in policies", func() {
			authorized, err := policyGuard.CheckAccess(policies, tokenData)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
			Expect(fakeCCClient.GetSpaceGUIDsCallCount()).To(Equal(1))
			token, appGUIDs := fakeCCClient.GetSpaceGUIDsArgsForCall(0)
			Expect(token).To(Equal("policy-server-token"))
			Expect(appGUIDs).To(ConsistOf([]string{"some-app-guid", "some-other-guid", "yet-another-guid"}))
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
			Expect(fakeCCClient.GetSubjectSpaceCallCount()).To(Equal(3))
			token, subjectId, checkSubjectSpace := fakeCCClient.GetSubjectSpaceArgsForCall(0)
			Expect(token).To(Equal("policy-server-token"))
			Expect(subjectId).To(Equal("some-developer-guid"))
			Expect(checkSubjectSpace).To(Equal(space1))
			token, subjectId, checkSubjectSpace = fakeCCClient.GetSubjectSpaceArgsForCall(1)
			Expect(token).To(Equal("policy-server-token"))
			Expect(subjectId).To(Equal("some-developer-guid"))
			Expect(checkSubjectSpace).To(Equal(space2))
			token, subjectId, checkSubjectSpace = fakeCCClient.GetSubjectSpaceArgsForCall(2)
			Expect(token).To(Equal("policy-server-token"))
			Expect(subjectId).To(Equal("some-developer-guid"))
			Expect(checkSubjectSpace).To(Equal(space3))
			Expect(authorized).To(BeTrue())
		})

		Context("when the token has a client as the subject", func() {
			BeforeEach(func() {
				tokenData = uaa_client.CheckTokenResponse{
					ClientID: "some-client-id",
					Subject:  "some-client-id",
				}
			})

			It("checks that the client can access all apps references in policies", func() {
				authorized, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
				Expect(fakeCCClient.GetSpaceGUIDsCallCount()).To(Equal(1))
				token, appGUIDs := fakeCCClient.GetSpaceGUIDsArgsForCall(0)
				Expect(token).To(Equal("policy-server-token"))
				Expect(appGUIDs).To(ConsistOf([]string{"some-app-guid", "some-other-guid", "yet-another-guid"}))
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
				Expect(fakeCCClient.GetSubjectSpaceCallCount()).To(Equal(3))
				token, subjectId, checkSubjectSpace := fakeCCClient.GetSubjectSpaceArgsForCall(0)
				Expect(token).To(Equal("policy-server-token"))
				Expect(subjectId).To(Equal("some-client-id"))
				Expect(checkSubjectSpace).To(Equal(space1))
				token, subjectId, checkSubjectSpace = fakeCCClient.GetSubjectSpaceArgsForCall(1)
				Expect(token).To(Equal("policy-server-token"))
				Expect(subjectId).To(Equal("some-client-id"))
				Expect(checkSubjectSpace).To(Equal(space2))
				token, subjectId, checkSubjectSpace = fakeCCClient.GetSubjectSpaceArgsForCall(2)
				Expect(token).To(Equal("policy-server-token"))
				Expect(subjectId).To(Equal("some-client-id"))
				Expect(checkSubjectSpace).To(Equal(space3))
				Expect(authorized).To(BeTrue())
			})
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
				Expect(fakeCCClient.GetSpaceCallCount()).To(Equal(0))
				Expect(fakeCCClient.GetSubjectSpaceCallCount()).To(Equal(0))
				Expect(authorized).To(BeTrue())
			})
		})

		Context("when the getting one of the the spaces returns nil", func() {
			BeforeEach(func() {
				fakeCCClient.GetSpaceReturns(nil, nil)
			})
			It("returns false", func() {
				authorized, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).NotTo(HaveOccurred())
				Expect(authorized).To(BeFalse())
			})
		})

		Context("when the getting the subjects spaces returns nil", func() {
			BeforeEach(func() {
				fakeCCClient.GetSubjectSpaceReturns(nil, nil)
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
				authorized, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting token: banana"))
				Expect(authorized).To(BeFalse())
			})
		})

		Context("when the getting the space guids fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetSpaceGUIDsReturns(nil, errors.New("banana"))
			})
			It("returns a useful error", func() {
				authorized, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting space guids: banana"))
				Expect(authorized).To(BeFalse())
			})
		})

		Context("when the getting one of the the spaces fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetSpaceReturns(nil, errors.New("banana"))
			})
			It("returns a useful error", func() {
				authorized, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting space with guid space-guid-1: banana"))
				Expect(authorized).To(BeFalse())
			})
		})

		Context("when the getting the subjects spaces fails", func() {
			BeforeEach(func() {
				fakeCCClient.GetSubjectSpaceReturns(nil, errors.New("banana"))
			})
			It("returns a useful error", func() {
				authorized, err := policyGuard.CheckAccess(policies, tokenData)
				Expect(err).To(MatchError("getting space with guid space-guid-1: banana"))
				Expect(authorized).To(BeFalse())
			})
		})
	})
})
