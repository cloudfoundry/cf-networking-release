package cleaner_test

import (
	"errors"

	"code.cloudfoundry.org/lager/lagertest"
	ccfakes "code.cloudfoundry.org/policy-server/cc_client/fakes"
	"code.cloudfoundry.org/policy-server/cleaner"
	"code.cloudfoundry.org/policy-server/cleaner/fakes"
	"code.cloudfoundry.org/policy-server/store"
	uaafakes "code.cloudfoundry.org/policy-server/uaa_client/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("PolicyCleaner", func() {
	var (
		policyCleaner *cleaner.PolicyCleaner
		fakeStore     *fakes.PolicyStore
		fakeUAAClient *uaafakes.UAAClient
		fakeCCClient  *ccfakes.CCClient
		logger        *lagertest.TestLogger
		c2cPolicies   []store.Policy
	)

	BeforeEach(func() {
		c2cPolicies = []store.Policy{{
			Source: store.Source{ID: "live-guid", Tag: "tag"},
			Destination: store.Destination{
				ID:       "live-guid",
				Tag:      "tag",
				Protocol: "tcp",
				Ports: store.Ports{
					Start: 8080,
					End:   8080,
				},
			},
		}, {
			Source: store.Source{ID: "dead-guid", Tag: "tag"},
			Destination: store.Destination{
				ID:       "live-guid",
				Tag:      "tag",
				Protocol: "udp",
				Ports: store.Ports{
					Start: 1234,
					End:   1234,
				},
			},
		}, {
			Source: store.Source{ID: "live-guid", Tag: "tag"},
			Destination: store.Destination{
				ID:       "dead-guid",
				Tag:      "tag",
				Protocol: "udp",
				Ports: store.Ports{
					Start: 1234,
					End:   1234,
				},
			},
		}}

		fakeStore = &fakes.PolicyStore{}
		fakeUAAClient = &uaafakes.UAAClient{}
		fakeCCClient = &ccfakes.CCClient{}
		logger = lagertest.NewTestLogger("test")
		policyCleaner = cleaner.NewPolicyCleaner(logger, fakeStore, fakeUAAClient, fakeCCClient, 0)

		fakeUAAClient.GetTokenReturns("valid-token", nil)
		fakeStore.AllReturns(c2cPolicies, nil)
		fakeCCClient.GetLiveSpaceGUIDsReturns(map[string]struct{}{"live-egress-space-guid": {}}, nil)
		fakeCCClient.GetLiveAppGUIDsStub = func(token string, appGUIDs []string) (map[string]struct{}, error) {
			liveGUIDs := make(map[string]struct{})
			for _, guid := range appGUIDs {
				if guid == "live-guid" || guid == "live-egress-app-guid" {
					liveGUIDs[guid] = struct{}{}
				}
			}
			return liveGUIDs, nil
		}
	})

	It("Deletes c2c and egress policies that reference apps that do not exist", func() {
		deletedPolicies, err := policyCleaner.DeleteStalePolicies()
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeStore.AllCallCount()).To(Equal(1))
		Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
		Expect(fakeCCClient.GetLiveAppGUIDsCallCount()).To(Equal(1))
		token, guids := fakeCCClient.GetLiveAppGUIDsArgsForCall(0)
		Expect(token).To(Equal("valid-token"))
		Expect(guids).To(ConsistOf("live-guid", "dead-guid"))

		stalePolicies := c2cPolicies[1:]

		Expect(fakeStore.DeleteCallCount()).To(Equal(1))
		Expect(fakeStore.DeleteArgsForCall(0)).To(Equal(stalePolicies))

		Expect(logger).To(gbytes.Say("deleting stale policies:.*c2c_policies.*dead-guid.*dead-guid.*total_c2c_policies\":2"))
		Expect(deletedPolicies).To(Equal(stalePolicies))
	})

	Context("when there are more apps with policies than the CC chunk size", func() {
		BeforeEach(func() {
			policyCleaner = &cleaner.PolicyCleaner{
				Logger:                logger,
				Store:                 fakeStore,
				UAAClient:             fakeUAAClient,
				CCClient:              fakeCCClient,
				CCAppRequestChunkSize: 1,
			}
		})

		It("Calls the CC server multiple times to check which policies to delete", func() {
			returnedPolicies, err := policyCleaner.DeleteStalePolicies()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStore.AllCallCount()).To(Equal(1))
			Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
			Expect(fakeCCClient.GetLiveAppGUIDsCallCount()).To(Equal(2))
			token0, guids0 := fakeCCClient.GetLiveAppGUIDsArgsForCall(0)
			Expect(token0).To(Equal("valid-token"))
			token1, guids1 := fakeCCClient.GetLiveAppGUIDsArgsForCall(1)
			Expect(token1).To(Equal("valid-token"))
			Expect([][]string{guids0, guids1}).To(ConsistOf(
				[]string{"live-guid"},
				[]string{"dead-guid"},
			))

			stalePolicies := c2cPolicies[1:]
			Expect(fakeStore.DeleteCallCount()).To(Equal(1))

			deletedPolicies := fakeStore.DeleteArgsForCall(0)
			Expect(deletedPolicies).To(Equal(stalePolicies))

			Expect(logger).To(gbytes.Say("deleting stale policies:.*c2c_policies.*dead-guid.*dead-guid.*total_c2c_policies\":2"))

			staleAPIPolicies := c2cPolicies[1:]
			Expect(returnedPolicies).To(ConsistOf(staleAPIPolicies[0], staleAPIPolicies[1]))
		})
	})

	Context("When retrieving policies from the db fails", func() {
		BeforeEach(func() {
			fakeStore.AllReturns([]store.Policy{}, errors.New("potato"))
		})

		It("returns a meaningful error", func() {
			_, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("database read failed for c2c policies: potato"))
		})

		It("logs the error", func() {
			policyCleaner.DeleteStalePolicies()
			Expect(logger).To(gbytes.Say("store-list-policies-failed.*potato"))
		})
	})

	Context("When getting the UAA token fails", func() {
		BeforeEach(func() {
			fakeUAAClient.GetTokenReturns("", errors.New("potato"))
		})

		It("returns a meaningful error", func() {
			_, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("get UAA token failed: potato"))
		})

		It("logs the full error", func() {
			policyCleaner.DeleteStalePolicies()
			Expect(logger).To(gbytes.Say("get-uaa-token-failed.*potato"))
		})
	})

	Context("When getting the apps from the Cloud-Controller fails", func() {
		BeforeEach(func() {
			fakeCCClient.GetLiveAppGUIDsReturns(nil, errors.New("potato"))
		})

		It("returns a meaningful error", func() {
			_, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("get app guids from Cloud-Controller failed: potato"))
		})

		It("logs the full error", func() {
			policyCleaner.DeleteStalePolicies()
			Expect(logger).To(gbytes.Say("cc-get-app-guids-failed.*potato"))
		})
	})

	Context("When deleting the policies fails", func() {
		BeforeEach(func() {
			fakeStore.DeleteReturns(errors.New("potato"))
		})

		It("returns a meaningful error", func() {
			_, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("database write failed: potato"))
		})

		It("logs the full error", func() {
			policyCleaner.DeleteStalePolicies()
			Expect(logger).To(gbytes.Say("store-delete-policies-failed.*potato"))
		})
	})
})
