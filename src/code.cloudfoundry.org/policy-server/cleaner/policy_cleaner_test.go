package cleaner_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/policy-server/cleaner"
	"code.cloudfoundry.org/policy-server/cleaner/fakes"
	"code.cloudfoundry.org/policy-server/store"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("PolicyCleaner", func() {
	var (
		policyCleaner   *cleaner.PolicyCleaner
		fakeStore       *fakes.PolicyStore
		fakeEgressStore *fakes.EgressPolicyStore
		fakeUAAClient   *fakes.UAAClient
		fakeCCClient    *fakes.CCClient
		logger          *lagertest.TestLogger
		c2cPolicies     []store.Policy
		egressPolicies  []store.EgressPolicy
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

		egressPolicies = []store.EgressPolicy{{
			ID:     "live-egress-policy-guid-1",
			Source: store.EgressSource{ID: "live-egress-app-guid", Type: "app"},
			Destination: store.EgressDestination{
				Rules: []store.EgressDestinationRule{
					{
						Protocol: "tcp",
						Ports: []store.Ports{
							{
								Start: 8080,
								End:   8080,
							},
						},
						IPRanges: []store.IPRange{
							{
								Start: "1.2.3.4",
								End:   "1.2.3.4",
							},
						},
					},
				},
			},
		}, {
			ID:     "live-egress-policy-guid-2",
			Source: store.EgressSource{ID: "live-egress-space-guid", Type: "space"},
			Destination: store.EgressDestination{
				Rules: []store.EgressDestinationRule{
					{
						Protocol: "tcp",
						Ports: []store.Ports{
							{
								Start: 8080,
								End:   8080,
							},
						},
						IPRanges: []store.IPRange{
							{
								Start: "1.2.3.4",
								End:   "1.2.3.4",
							},
						},
					},
				},
			},
		}, {
			ID:     "dead-egress-policy-guid-3",
			Source: store.EgressSource{ID: "dead-egress-app-guid", Type: "app"},
			Destination: store.EgressDestination{
				Rules: []store.EgressDestinationRule{
					{
						Protocol: "tcp",
						Ports: []store.Ports{
							{
								Start: 8080,
								End:   8080,
							},
						},
						IPRanges: []store.IPRange{
							{
								Start: "1.2.3.4",
								End:   "1.2.3.4",
							},
						},
					},
				},
			},
		}, {
			ID:     "dead-egress-policy-guid-4",
			Source: store.EgressSource{ID: "dead-egress-space-guid", Type: "space"},
			Destination: store.EgressDestination{
				Rules: []store.EgressDestinationRule{
					{
						Protocol: "tcp",
						Ports: []store.Ports{
							{
								Start: 8080,
								End:   8080,
							},
						},
						IPRanges: []store.IPRange{
							{
								Start: "1.2.3.4",
								End:   "1.2.3.4",
							},
						},
					},
				},
			},
		},
		}

		fakeStore = &fakes.PolicyStore{}
		fakeEgressStore = &fakes.EgressPolicyStore{}
		fakeUAAClient = &fakes.UAAClient{}
		fakeCCClient = &fakes.CCClient{}
		logger = lagertest.NewTestLogger("test")
		policyCleaner = cleaner.NewPolicyCleaner(logger, fakeStore, fakeEgressStore, fakeUAAClient, fakeCCClient, 0, 5*time.Second)

		fakeUAAClient.GetTokenReturns("valid-token", nil)
		fakeStore.AllReturns(c2cPolicies, nil)
		fakeEgressStore.AllReturns(egressPolicies, nil)
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
		deletedPolicies, deletedEgressPolicies, err := policyCleaner.DeleteStalePolicies()
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeStore.AllCallCount()).To(Equal(1))
		Expect(fakeEgressStore.AllCallCount()).To(Equal(1))
		Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
		Expect(fakeCCClient.GetLiveSpaceGUIDsCallCount()).To(Equal(1))
		token0, guids0 := fakeCCClient.GetLiveSpaceGUIDsArgsForCall(0)
		Expect(token0).To(Equal("valid-token"))
		Expect(guids0).To(ConsistOf(
			"live-egress-space-guid",
			"dead-egress-space-guid",
		))
		Expect(fakeCCClient.GetLiveAppGUIDsCallCount()).To(Equal(2))
		token, guids := fakeCCClient.GetLiveAppGUIDsArgsForCall(0)
		Expect(token).To(Equal("valid-token"))
		Expect(guids).To(ConsistOf("live-guid", "dead-guid"))

		_, guids = fakeCCClient.GetLiveAppGUIDsArgsForCall(1)
		Expect(guids).To(ConsistOf("live-egress-app-guid", "dead-egress-app-guid"))

		stalePolicies := c2cPolicies[1:]
		staleEgressPolicies := egressPolicies[2:]

		Expect(fakeStore.DeleteCallCount()).To(Equal(1))
		Expect(fakeStore.DeleteArgsForCall(0)).To(Equal(stalePolicies))

		Expect(fakeEgressStore.DeleteCallCount()).To(Equal(1))
		Expect(fakeEgressStore.DeleteArgsForCall(0)).To(Equal([]string{"dead-egress-policy-guid-3", "dead-egress-policy-guid-4"}))

		Expect(logger).To(gbytes.Say("deleting stale policies:.*c2c_policies.*dead-guid.*dead-guid.*egress_policies.*dead-egress-app-guid.*dead-egress-space-guid.*total_c2c_policies\":2.*total_egress_policies\":2"))
		Expect(deletedPolicies).To(Equal(stalePolicies))
		Expect(deletedEgressPolicies).To(Equal(staleEgressPolicies))
	})

	Context("when there are more apps with policies than the CC chunk size", func() {
		BeforeEach(func() {
			policyCleaner = &cleaner.PolicyCleaner{
				Logger:                logger,
				Store:                 fakeStore,
				EgressStore:           fakeEgressStore,
				UAAClient:             fakeUAAClient,
				CCClient:              fakeCCClient,
				CCAppRequestChunkSize: 1,
				RequestTimeout:        time.Duration(5) * time.Second,
			}
		})

		It("Calls the CC server multiple times to check which policies to delete", func() {
			returnedPolicies, _, err := policyCleaner.DeleteStalePolicies()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStore.AllCallCount()).To(Equal(1))
			Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
			Expect(fakeCCClient.GetLiveAppGUIDsCallCount()).To(Equal(4))
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

	It("returns a helpful error when get live space guids call fails", func() {
		fakeCCClient.GetLiveSpaceGUIDsReturns(nil, errors.New("yankee"))

		_, _, err := policyCleaner.DeleteStalePolicies()
		Expect(err).To(MatchError("get live space guids failed: yankee"))
		Expect(logger).To(gbytes.Say("get-live-space-guids-failed.*yankee"))
	})

	Context("When retrieving policies from the db fails", func() {
		BeforeEach(func() {
			fakeStore.AllReturns([]store.Policy{}, errors.New("potato"))
		})

		It("returns a meaningful error", func() {
			_, _, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("database read failed for c2c policies: potato"))
		})

		It("logs the error", func() {
			policyCleaner.DeleteStalePolicies()
			Expect(logger).To(gbytes.Say("store-list-policies-failed.*potato"))
		})
	})

	Context("When retrieving egress policies from the db fails", func() {
		BeforeEach(func() {
			fakeEgressStore.AllReturns([]store.EgressPolicy{}, errors.New("potato"))
		})

		It("returns a meaningful error", func() {
			_, _, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("database read failed for egress policies: potato"))
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
			_, _, err := policyCleaner.DeleteStalePolicies()
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
			_, _, err := policyCleaner.DeleteStalePolicies()
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
			_, _, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("database write failed: potato"))
		})

		It("logs the full error", func() {
			policyCleaner.DeleteStalePolicies()
			Expect(logger).To(gbytes.Say("store-delete-policies-failed.*potato"))
		})
	})

	Context("When deleting the egress policies fails", func() {
		BeforeEach(func() {
			fakeEgressStore.DeleteReturns(nil, errors.New("potato"))
		})

		It("returns a meaningful error", func() {
			_, _, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("database write failed: potato"))
		})

		It("logs the full error", func() {
			policyCleaner.DeleteStalePolicies()
			Expect(logger).To(gbytes.Say("store-delete-policies-failed.*potato"))
		})
	})
})
