package store_test

import (
	"errors"
	dbfakes "policy-server/db/fakes"
	"policy-server/store"
	"policy-server/store/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressPolicyStore", func() {
	var (
		egressPolicyStore *store.EgressPolicyStore
		egressPolicyRepo  *fakes.EgressPolicyRepo

		tx             *dbfakes.Transaction
		egressPolicies []store.EgressPolicy
		spacePolicy    store.EgressPolicy
	)

	BeforeEach(func() {
		egressPolicyRepo = &fakes.EgressPolicyRepo{}
		egressPolicyStore = &store.EgressPolicyStore{
			EgressPolicyRepo: egressPolicyRepo,
		}

		tx = &dbfakes.Transaction{}
		egressPolicies = []store.EgressPolicy{
			{
				Source: store.EgressSource{
					ID: "some-app-guid",
				},
				Destination: store.EgressDestination{
					Protocol: "tcp",
					Ports: []store.Ports{
						{
							Start: 8080,
							End:   8081,
						},
					},
					IPRanges: []store.IPRange{
						{
							Start: "1.2.3.4",
							End:   "1.2.3.5",
						},
					},
				},
			},
			{
				Source: store.EgressSource{
					ID: "different-app-guid",
				},
				Destination: store.EgressDestination{
					Protocol: "udp",
					IPRanges: []store.IPRange{
						{
							Start: "2.2.3.4",
							End:   "2.2.3.5",
						},
					},
				},
			},
			{
				Source: store.EgressSource{
					ID: "different-app-guid",
				},
				Destination: store.EgressDestination{
					Protocol: "icmp",
					IPRanges: []store.IPRange{
						{
							Start: "2.2.3.4",
							End:   "2.2.3.5",
						},
					},
					ICMPType: 1,
					ICMPCode: 2,
				},
			},
		}

		spacePolicy = store.EgressPolicy{
			Source: store.EgressSource{
				Type: "space",
				ID:   "space-guid",
			},
			Destination: store.EgressDestination{
				Protocol: "icmp",
				IPRanges: []store.IPRange{
					{
						Start: "3.2.3.4",
						End:   "3.2.3.5",
					},
				},
				ICMPType: 2,
				ICMPCode: 3,
			},
		}

		egressPolicyRepo.GetTerminalByAppGUIDReturns(-1, nil)
	})

	Describe("CreateWithTx", func() {
		It("creates a source and destination terminal", func() {
			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateTerminalCallCount()).To(Equal(6))
			Expect(egressPolicyRepo.CreateTerminalArgsForCall(0)).To(Equal(tx))
			Expect(egressPolicyRepo.CreateTerminalArgsForCall(1)).To(Equal(tx))
			Expect(egressPolicyRepo.CreateTerminalArgsForCall(2)).To(Equal(tx))
			Expect(egressPolicyRepo.CreateTerminalArgsForCall(3)).To(Equal(tx))
		})

		It("returns an error when CreateTerminal fails", func() {
			egressPolicyRepo.CreateTerminalReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).To(MatchError("failed to create source terminal: OMG WHY DID THIS FAIL"))
		})

		It("creates an app with the sourceTerminalID", func() {
			egressPolicyRepo.CreateTerminalReturnsOnCall(0, 42, nil)
			egressPolicyRepo.CreateTerminalReturnsOnCall(2, 24, nil)

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).NotTo(HaveOccurred())

			Expect(egressPolicyRepo.CreateAppCallCount()).To(Equal(3))
			argTx, argSourceTerminalId, argAppGUID := egressPolicyRepo.CreateAppArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(argSourceTerminalId).To(Equal(int64(42)))
			Expect(argAppGUID).To(Equal("some-app-guid"))

			argTx, argSourceTerminalId, argAppGUID = egressPolicyRepo.CreateAppArgsForCall(1)
			Expect(argTx).To(Equal(tx))
			Expect(argSourceTerminalId).To(Equal(int64(24)))
			Expect(argAppGUID).To(Equal("different-app-guid"))

			Expect(egressPolicyRepo.CreateSpaceCallCount()).To(Equal(0))
		})

		It("returns an error when the CreateApp fails", func() {
			egressPolicyRepo.CreateAppReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).To(MatchError("failed to create source app: OMG WHY DID THIS FAIL"))
		})

		It("creates a space with a sourceTerminalID", func() {
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns(-1, nil)
			egressPolicyRepo.CreateTerminalReturns(66, nil)
			err := egressPolicyStore.CreateWithTx(tx, []store.EgressPolicy{spacePolicy})
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateSpaceCallCount()).To(Equal(1))
			Expect(egressPolicyRepo.CreateAppCallCount()).To(Equal(0))
			argTx, argSourceTerminalID, argSpaceGUID := egressPolicyRepo.CreateSpaceArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(argSourceTerminalID).To(Equal(int64(66)))
			Expect(argSpaceGUID).To(Equal("space-guid"))
		})

		It("creates an ip range with the destinationTerminalID", func() {
			egressPolicyRepo.CreateTerminalReturnsOnCall(1, 42, nil)
			egressPolicyRepo.CreateTerminalReturnsOnCall(3, 24, nil)
			egressPolicyRepo.CreateTerminalReturnsOnCall(5, 44, nil)

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateIPRangeCallCount()).To(Equal(3))

			argTx, destinationID, startIP, endIP, protocol, startPort, endPort, icmpType, icmpCode := egressPolicyRepo.CreateIPRangeArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(destinationID).To(Equal(int64(42)))
			Expect(startPort).To(Equal(int64(8080)))
			Expect(endPort).To(Equal(int64(8081)))
			Expect(startIP).To(Equal("1.2.3.4"))
			Expect(endIP).To(Equal("1.2.3.5"))
			Expect(protocol).To(Equal("tcp"))
			Expect(icmpType).To(Equal(int64(0)))
			Expect(icmpCode).To(Equal(int64(0)))

			argTx, destinationID, startIP, endIP, protocol, startPort, endPort, icmpType, icmpCode = egressPolicyRepo.CreateIPRangeArgsForCall(1)
			Expect(argTx).To(Equal(tx))
			Expect(destinationID).To(Equal(int64(24)))
			Expect(startPort).To(Equal(int64(0)))
			Expect(endPort).To(Equal(int64(0)))
			Expect(startIP).To(Equal("2.2.3.4"))
			Expect(endIP).To(Equal("2.2.3.5"))
			Expect(protocol).To(Equal("udp"))
			Expect(icmpType).To(Equal(int64(0)))
			Expect(icmpCode).To(Equal(int64(0)))

			argTx, destinationID, startIP, endIP, protocol, startPort, endPort, icmpType, icmpCode = egressPolicyRepo.CreateIPRangeArgsForCall(2)
			Expect(argTx).To(Equal(tx))
			Expect(destinationID).To(Equal(int64(44)))
			Expect(startPort).To(Equal(int64(0)))
			Expect(endPort).To(Equal(int64(0)))
			Expect(startIP).To(Equal("2.2.3.4"))
			Expect(endIP).To(Equal("2.2.3.5"))
			Expect(protocol).To(Equal("icmp"))
			Expect(icmpType).To(Equal(int64(1)))
			Expect(icmpCode).To(Equal(int64(2)))
		})

		It("returns an error when the CreateIPRange fails", func() {
			egressPolicyRepo.CreateIPRangeReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).To(MatchError("failed to create ip range: OMG WHY DID THIS FAIL"))
		})

		It("creates an egress policy with the right IDs", func() {
			egressPolicyRepo.CreateTerminalReturnsOnCall(0, 11, nil)
			egressPolicyRepo.CreateTerminalReturnsOnCall(1, 22, nil)
			egressPolicyRepo.CreateTerminalReturnsOnCall(2, 33, nil)
			egressPolicyRepo.CreateTerminalReturnsOnCall(3, 44, nil)

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateEgressPolicyCallCount()).To(Equal(3))

			argTx, sourceID, destinationID := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal(int64(11)))
			Expect(destinationID).To(Equal(int64(22)))

			argTx, sourceID, destinationID = egressPolicyRepo.CreateEgressPolicyArgsForCall(1)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal(int64(33)))
			Expect(destinationID).To(Equal(int64(44)))
		})

		It("returns an error when the CreateEgressPolicy fails", func() {
			egressPolicyRepo.CreateEgressPolicyReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).To(MatchError("failed to create egress policy: OMG WHY DID THIS FAIL"))
		})

		It("uses the existing app terminal id when it exists", func() {
			egressPolicyRepo.GetTerminalByAppGUIDReturns(66, nil)

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateAppCallCount()).To(Equal(0))
			_, sourceID, _ := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(sourceID).To(Equal(int64(66)))
		})

		It("uses the existing space terminal id when it exists", func() {
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns(55, nil)

			err := egressPolicyStore.CreateWithTx(tx, []store.EgressPolicy{spacePolicy})
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateSpaceCallCount()).To(Equal(0))
			_, sourceID, _ := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(sourceID).To(Equal(int64(55)))
		})

		It("returns an error when the CreateTerminal fails for space", func() {
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns(-1, nil)
			egressPolicyRepo.CreateTerminalReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, []store.EgressPolicy{spacePolicy})
			Expect(err).To(MatchError("failed to create source terminal: OMG WHY DID THIS FAIL"))
		})

		It("returns an error when the CreateSpace fails", func() {
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns(-1, nil)
			egressPolicyRepo.CreateSpaceReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, []store.EgressPolicy{spacePolicy})
			Expect(err).To(MatchError("failed to create space: OMG WHY DID THIS FAIL"))
		})

		It("returns an error when the GetTerminalBySpaceGUID fails", func() {
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, []store.EgressPolicy{spacePolicy})
			Expect(err).To(MatchError("failed to get terminal by space guid: OMG WHY DID THIS FAIL"))
		})

		It("returns an error when the GetTerminalByAppGUID fails", func() {
			egressPolicyRepo.GetTerminalByAppGUIDReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			err := egressPolicyStore.CreateWithTx(tx, egressPolicies)
			Expect(err).To(MatchError("failed to get terminal by app guid: OMG WHY DID THIS FAIL"))
		})
	})

	Describe("DeleteWithTx", func() {
		var (
			egressPoliciesToDelete   []store.EgressPolicy
			egressPolicyIDCollection store.EgressPolicyIDCollection
			egressPolicyID           int64
			ipRangeID                int64
			destTerminalID           int64
			appID                    int64
			srcTerminalID            int64
		)
		BeforeEach(func() {
			egressPoliciesToDelete = []store.EgressPolicy{
				{
					Source: store.EgressSource{
						ID: "some-app-guid",
					},
					Destination: store.EgressDestination{
						Protocol: "tcp",
						Ports: []store.Ports{
							{
								Start: 8080,
								End:   8081,
							},
						},
						IPRanges: []store.IPRange{
							{
								Start: "1.2.3.4",
								End:   "1.2.3.5",
							},
						},
					},
				},
			}

			egressPolicyID = 6
			ipRangeID = 9
			destTerminalID = 12
			appID = 21
			srcTerminalID = 22

			egressPolicyIDCollection = store.EgressPolicyIDCollection{
				EgressPolicyID:        egressPolicyID,
				DestinationIPRangeID:  ipRangeID,
				DestinationTerminalID: destTerminalID,
				SourceAppID:           appID,
				SourceSpaceID:         -1,
				SourceTerminalID:      srcTerminalID,
			}
			egressPolicyRepo.GetIDsByEgressPolicyReturns(egressPolicyIDCollection, nil)
		})

		It("deletes the egress policy", func() {
			err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
			Expect(err).NotTo(HaveOccurred())

			Expect(egressPolicyRepo.GetIDsByEgressPolicyCallCount()).To(Equal(1))
			passedTx, passedEgressPolicy := egressPolicyRepo.GetIDsByEgressPolicyArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedEgressPolicy).To(Equal(egressPoliciesToDelete[0]))

			Expect(egressPolicyRepo.DeleteEgressPolicyCallCount()).To(Equal(1))
			passedTx, passedEgressPolicyID := egressPolicyRepo.DeleteEgressPolicyArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedEgressPolicyID).To(Equal(egressPolicyID))

			Expect(egressPolicyRepo.DeleteIPRangeCallCount()).To(Equal(1))
			passedTx, passedIPRangeID := egressPolicyRepo.DeleteIPRangeArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedIPRangeID).To(Equal(ipRangeID))

			Expect(egressPolicyRepo.DeleteTerminalCallCount()).To(Equal(2))
			passedTx, passedDestTerminalID := egressPolicyRepo.DeleteTerminalArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedDestTerminalID).To(Equal(destTerminalID))

			Expect(egressPolicyRepo.DeleteAppCallCount()).To(Equal(1))
			passedTx, passedAppID := egressPolicyRepo.DeleteAppArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedAppID).To(Equal(appID))

			passedTx, passedSrcTerminalID := egressPolicyRepo.DeleteTerminalArgsForCall(1)
			Expect(passedTx).To(Equal(tx))
			Expect(passedSrcTerminalID).To(Equal(srcTerminalID))

			Expect(egressPolicyRepo.DeleteSpaceCallCount()).To(Equal(0))
		})

		Context("when the source terminal is attached to a space", func() {
			var (
				spaceID int64
			)

			BeforeEach(func() {
				spaceID = 23
				egressPolicyIDCollection.SourceAppID = -1
				egressPolicyIDCollection.SourceSpaceID = spaceID
				egressPolicyRepo.GetIDsByEgressPolicyReturns(egressPolicyIDCollection, nil)
			})

			It("deletes the space", func() {
				err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
				Expect(err).NotTo(HaveOccurred())

				Expect(egressPolicyRepo.DeleteAppCallCount()).To(Equal(0))
				Expect(egressPolicyRepo.DeleteSpaceCallCount()).To(Equal(1))
				passedTx, passedSpaceID := egressPolicyRepo.DeleteSpaceArgsForCall(0)
				Expect(passedTx).To(Equal(tx))
				Expect(passedSpaceID).To(Equal(spaceID))
			})

			Context("when the EgressPolicyRepo.DeleteSpace fails", func() {
				BeforeEach(func() {
					egressPolicyRepo.DeleteSpaceReturns(errors.New("ther's a bug"))
				})

				It("returns an error", func() {
					err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
					Expect(err).To(MatchError("failed to delete source space: ther's a bug"))
				})
			})
		})

		Context("when there are multiple egress policies", func() {
			BeforeEach(func() {
				egressPoliciesToDelete = append(egressPoliciesToDelete, store.EgressPolicy{
					Source: store.EgressSource{
						ID: "some-other-app-guid",
					},
					Destination: store.EgressDestination{
						Protocol: "tcp",
						Ports: []store.Ports{
							{
								Start: 8080,
								End:   8081,
							},
						},
						IPRanges: []store.IPRange{
							{
								Start: "1.2.3.4",
								End:   "1.2.3.6",
							},
						},
					},
				})
			})

			It("deletes all the egress policies", func() {
				err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
				Expect(err).NotTo(HaveOccurred())

				Expect(egressPolicyRepo.GetIDsByEgressPolicyCallCount()).To(Equal(2))
				passedTx, passedEgressPolicy := egressPolicyRepo.GetIDsByEgressPolicyArgsForCall(0)
				Expect(passedTx).To(Equal(tx))
				Expect(passedEgressPolicy).To(Equal(egressPoliciesToDelete[0]))

				passedTx, passedEgressPolicy = egressPolicyRepo.GetIDsByEgressPolicyArgsForCall(1)
				Expect(passedTx).To(Equal(tx))
				Expect(passedEgressPolicy).To(Equal(egressPoliciesToDelete[1]))

				Expect(egressPolicyRepo.DeleteEgressPolicyCallCount()).To(Equal(2))
				Expect(egressPolicyRepo.DeleteIPRangeCallCount()).To(Equal(2))
				Expect(egressPolicyRepo.DeleteTerminalCallCount()).To(Equal(4))
				Expect(egressPolicyRepo.DeleteAppCallCount()).To(Equal(2))
			})
		})

		Context("when app is referenced by another egress policy", func() {
			BeforeEach(func() {
				egressPolicyRepo.IsTerminalInUseReturns(true, nil)
			})

			It("doesn't delete the source terminal or source app", func() {
				err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
				Expect(err).NotTo(HaveOccurred())

				Expect(egressPolicyRepo.IsTerminalInUseCallCount()).To(Equal(1))
				passedTx, passedTerminalID := egressPolicyRepo.IsTerminalInUseArgsForCall(0)
				Expect(passedTx).To(Equal(tx))
				Expect(passedTerminalID).To(Equal(srcTerminalID))

				Expect(egressPolicyRepo.DeleteAppCallCount()).To(Equal(0))

				Expect(egressPolicyRepo.DeleteTerminalCallCount()).To(Equal(1))
				passedTx, passedDestTerminalID := egressPolicyRepo.DeleteTerminalArgsForCall(0)
				Expect(passedTx).To(Equal(tx))
				Expect(passedDestTerminalID).To(Equal(destTerminalID))
			})
		})

		Context("when the EgressPolicyRepo.GetIDsByEgressPolicy fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetIDsByEgressPolicyReturns(store.EgressPolicyIDCollection{}, errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to find egress policy: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteEgressPolicy fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.DeleteEgressPolicyReturns(errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to delete egress policy: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteIPRange fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.DeleteIPRangeReturns(errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to delete destination ip range: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteTerminal fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.DeleteTerminalReturns(errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to delete destination terminal: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.IsTerminalInUse fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.IsTerminalInUseReturns(false, errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to check if source terminal is in use: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteApp fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.DeleteAppReturns(errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to delete source app: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteTerminal fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.DeleteTerminalReturnsOnCall(1, errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.DeleteWithTx(tx, egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to delete source terminal: ther's a bug"))
			})
		})
	})

	Describe("All", func() {
		Context("when there are policies created", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetAllPoliciesReturns(egressPolicies, nil)
			})

			It("should return a list of all policies", func() {
				policies, err := egressPolicyStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(Equal(egressPolicies))
			})
		})
	})

	Describe("ByGuids", func() {
		Context("when called with ids", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetByGuidsReturns(egressPolicies, nil)
			})

			It("calls egressPolicyRepo.GetByGuid", func() {
				policies, err := egressPolicyStore.ByGuids([]string{"meow"})
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(Equal(egressPolicies))

				ids := egressPolicyRepo.GetByGuidsArgsForCall(0)
				Expect(ids).To(Equal([]string{"meow"}))
			})
		})

		Context("when an error is returned from the repo", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetByGuidsReturns(nil, errors.New("bark bark"))
			})

			It("calls egressPolicyRepo.GetByGuid", func() {
				_, err := egressPolicyStore.ByGuids([]string{"meow"})
				Expect(err).To(MatchError("failed to get policies by guids: bark bark"))
			})
		})
	})
})
