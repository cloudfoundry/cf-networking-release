package cleaner_test

import (
	"errors"
	"policy-server/cleaner"
	"policy-server/cleaner/fakes"
	"time"

	"policy-server/store"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("PolicyCleaner", func() {
	var (
		policyCleaner *cleaner.PolicyCleaner
		fakeStore     *fakes.ListDeleteStore
		fakeUAAClient *fakes.UAAClient
		fakeCCClient  *fakes.CCClient
		logger        *lagertest.TestLogger
		allPolicies   []store.Policy
	)

	BeforeEach(func() {
		allPolicies = []store.Policy{{
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

		fakeStore = &fakes.ListDeleteStore{}
		fakeUAAClient = &fakes.UAAClient{}
		fakeCCClient = &fakes.CCClient{}
		logger = lagertest.NewTestLogger("test")

		policyCleaner = &cleaner.PolicyCleaner{
			Logger:         logger,
			Store:          fakeStore,
			UAAClient:      fakeUAAClient,
			CCClient:       fakeCCClient,
			RequestTimeout: 5 * time.Second,
		}

		fakeUAAClient.GetTokenReturns("valid-token", nil)
		fakeStore.AllReturns(store.PolicyCollection{Policies: allPolicies}, nil)
		fakeCCClient.GetLiveAppGUIDsStub = func(token string, appGUIDs []string) (map[string]struct{}, error) {
			liveGUIDs := make(map[string]struct{})
			for _, guid := range appGUIDs {
				if guid == "live-guid" {
					liveGUIDs["live-guid"] = struct{}{}
				}
			}
			return liveGUIDs, nil
		}
	})

	It("Deletes policies that reference apps that do not exist", func() {
		deletedPolicyCollection, err := policyCleaner.DeleteStalePolicies()
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeStore.AllCallCount()).To(Equal(1))
		Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
		Expect(fakeCCClient.GetLiveAppGUIDsCallCount()).To(Equal(1))
		token, guids := fakeCCClient.GetLiveAppGUIDsArgsForCall(0)
		Expect(token).To(Equal("valid-token"))
		Expect(guids).To(ConsistOf("live-guid", "dead-guid"))

		stalePolicies := allPolicies[1:]

		expectedCollection := store.PolicyCollection{Policies: stalePolicies}

		Expect(fakeStore.DeleteCallCount()).To(Equal(2))
		Expect(fakeStore.DeleteArgsForCall(0)).To(Equal(expectedCollection))

		Expect(logger).To(gbytes.Say("deleting stale policies:.*policies.*dead-guid.*dead-guid.*total_policies\":2"))
		staleAPIPolicies := allPolicies[1:]
		Expect(deletedPolicyCollection).To(Equal(store.PolicyCollection{Policies: staleAPIPolicies}))
	})

	Context("when there are more apps with policies than the CC chunk size", func() {
		BeforeEach(func() {
			policyCleaner = &cleaner.PolicyCleaner{
				Logger:                logger,
				Store:                 fakeStore,
				UAAClient:             fakeUAAClient,
				CCClient:              fakeCCClient,
				CCAppRequestChunkSize: 1,
				RequestTimeout:        time.Duration(5) * time.Second,
			}
		})
		It("Calls the CC server multiple times to check which policies to delete", func() {
			returnedPolicyCollection, err := policyCleaner.DeleteStalePolicies()
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

			stalePolicies := allPolicies[1:]
			Expect(fakeStore.DeleteCallCount()).To(Equal(3))

			var deleted [][]store.Policy
			deletedPolicyCollection := fakeStore.DeleteArgsForCall(0)
			deleted = append(deleted, deletedPolicyCollection.Policies)
			deletedPolicyCollection = fakeStore.DeleteArgsForCall(1)
			deleted = append(deleted, deletedPolicyCollection.Policies)
			Expect(deleted).To(ConsistOf(stalePolicies, []store.Policy{}))

			Expect(logger).To(gbytes.Say("deleting stale policies:.*policies.*dead-guid.*dead-guid.*total_policies\":2"))

			staleAPIPolicies := allPolicies[1:]
			Expect(returnedPolicyCollection.Policies).To(ConsistOf(staleAPIPolicies[0], staleAPIPolicies[1]))
		})
	})

	Context("when there are egress space policies", func() {
		var (
			allEgressPolicies store.PolicyCollection
		)

		BeforeEach(func() {
			allEgressPolicies = store.PolicyCollection{
				EgressPolicies: []store.EgressPolicy{{
					Source: store.EgressSource{ID: "live-guid", Type: "space"},
					Destination: store.EgressDestination{
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
				}, {
					Source: store.EgressSource{ID: "dead-guid", Type: "space"},
					Destination: store.EgressDestination{
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
				}},
			}
		})

		It("deletes policies that reference spaces that do not exist", func() {
			fakeStore.AllReturns(allEgressPolicies, nil)
			fakeCCClient.GetLiveSpaceGUIDsReturns(map[string]struct{}{"live-guid": {}}, nil)

			_, err := policyCleaner.DeleteStalePolicies()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStore.AllCallCount()).To(Equal(1))
			Expect(fakeUAAClient.GetTokenCallCount()).To(Equal(1))
			Expect(fakeCCClient.GetLiveSpaceGUIDsCallCount()).To(Equal(1))
			token0, guids0 := fakeCCClient.GetLiveSpaceGUIDsArgsForCall(0)
			Expect(token0).To(Equal("valid-token"))
			Expect(guids0).To(ConsistOf(
				"live-guid",
				"dead-guid",
			))
			Expect(fakeStore.DeleteCallCount()).To(Equal(1))
			passedDeletePolicies := fakeStore.DeleteArgsForCall(0)
			Expect(passedDeletePolicies.EgressPolicies).To(ConsistOf(allEgressPolicies.EgressPolicies[1]))
		})

		It("returns a helpful error when egress store fails", func() {
			fakeStore.AllReturns(store.PolicyCollection{}, errors.New("whiskey"))

			_, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("database read failed: whiskey"))
			Expect(logger).To(gbytes.Say("store-list-policies-failed.*whiskey"))
		})

		It("returns a helpful error when get live space guids call fails", func() {
			fakeCCClient.GetLiveSpaceGUIDsReturns(nil, errors.New("yankee"))

			_, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("get live space guids failed: yankee"))
			Expect(logger).To(gbytes.Say("get-live-space-guids-failed.*yankee"))
		})

		It("returns a helpful error when egress store delete fails", func() {
			fakeStore.DeleteReturns(errors.New("zulu"))

			_, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("database write failed: zulu"))
			Expect(logger).To(gbytes.Say("store-delete-policies-failed.*zulu"))
		})
	})

	Context("When retrieving policies from the db fails", func() {
		BeforeEach(func() {
			fakeStore.AllReturns(store.PolicyCollection{}, errors.New("potato"))
		})

		It("returns a meaningful error", func() {
			_, err := policyCleaner.DeleteStalePolicies()
			Expect(err).To(MatchError("database read failed: potato"))
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

	Context("when the context times out", func() {
		//TODO
	})
})
