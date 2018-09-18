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
		terminalsRepo     *fakes.TerminalsRepo
		mockDb            *fakes.Db

		tx             *dbfakes.Transaction
		egressPolicies []store.EgressPolicy
		spacePolicy    store.EgressPolicy
	)

	BeforeEach(func() {
		egressPolicyRepo = &fakes.EgressPolicyRepo{}
		terminalsRepo = &fakes.TerminalsRepo{}
		mockDb = &fakes.Db{}
		tx = &dbfakes.Transaction{}

		egressPolicyStore = &store.EgressPolicyStore{
			TerminalsRepo:    terminalsRepo,
			EgressPolicyRepo: egressPolicyRepo,
			Conn:             mockDb,
		}

		mockDb.BeginxReturns(tx, nil)

		egressPolicies = []store.EgressPolicy{
			{
				Source: store.EgressSource{
					ID: "some-app-guid",
				},
				Destination: store.EgressDestination{
					GUID: "some-destination-guid",
				},
			},
			{
				Source: store.EgressSource{
					ID: "different-app-guid",
				},
				Destination: store.EgressDestination{
					GUID: "some-destination-guid-2",
				},
			},
		}

		spacePolicy = store.EgressPolicy{
			Source: store.EgressSource{
				Type: "space",
				ID:   "space-guid",
			},
			Destination: store.EgressDestination{
				GUID: "some-destination-guid",
			},
		}

		egressPolicyRepo.GetTerminalByAppGUIDReturns("", nil)
	})

	Describe("Create", func() {
		It("creates an egress policy with the right GUIDs", func() {
			terminalsRepo.CreateReturnsOnCall(0, "some-app-guid", nil)
			terminalsRepo.CreateReturnsOnCall(1, "some-space-guid", nil)
			egressPolicyRepo.CreateEgressPolicyReturnsOnCall(0, "some-egress-policy-guid-1", nil)
			egressPolicyRepo.CreateEgressPolicyReturnsOnCall(1, "some-egress-policy-guid-2", nil)

			createdPolicies, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateEgressPolicyCallCount()).To(Equal(2))
			Expect(createdPolicies).To(HaveLen(2))
			Expect(createdPolicies).To(Equal([]store.EgressPolicy{
				{
					ID: "some-egress-policy-guid-1",
					Source: store.EgressSource{
						ID: "some-app-guid",
					},
					Destination: store.EgressDestination{
						GUID: "some-destination-guid",
					},
				},
				{
					ID: "some-egress-policy-guid-2",
					Source: store.EgressSource{
						ID: "different-app-guid",
					},
					Destination: store.EgressDestination{
						GUID: "some-destination-guid-2",
					},
				},
			}))

			argTx, sourceID, destinationID := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal("some-app-guid"))
			Expect(destinationID).To(Equal("some-destination-guid"))

			argTx, sourceID, destinationID = egressPolicyRepo.CreateEgressPolicyArgsForCall(1)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal("some-space-guid"))
			Expect(destinationID).To(Equal("some-destination-guid-2"))
		})

		It("returns an error when the database connection can't begin a transaction", func() {
			mockDb.BeginxReturns(nil, errors.New("potato"))
			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).To(MatchError("create transaction: potato"))
		})

		It("starts/commits transaction", func() {
			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockDb.BeginxCallCount()).To(Equal(1))
			Expect(tx.CommitCallCount()).To(Equal(1))
		})

		It("returns an error when begin transaction fails", func() {
			mockDb.BeginxReturns(nil, errors.New("failed to begin"))
			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).To(MatchError("create transaction: failed to begin"))
		})

		It("returns an error when commit transaction fails", func() {
			tx.CommitReturns(errors.New("failed to commit"))
			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).To(MatchError("commit transaction: failed to commit"))
		})

		It("rollsback the tx when the createWithTx fails", func() {
			egressPolicyRepo.CreateAppReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).To(MatchError("failed to create source app: OMG WHY DID THIS FAIL"))
			Expect(tx.RollbackCallCount()).To(Equal(1))
		})

		It("returns an error when CreateTerminal fails", func() {
			terminalsRepo.CreateReturns("", errors.New("OMG WHY DID THIS FAIL"))

			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).To(MatchError("failed to create source terminal: OMG WHY DID THIS FAIL"))
		})

		It("creates an app with the sourceTerminalGUID", func() {
			terminalsRepo.CreateReturnsOnCall(0, "some-term-guid", nil)
			terminalsRepo.CreateReturnsOnCall(1, "some-term-guid-2", nil)

			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).NotTo(HaveOccurred())

			Expect(egressPolicyRepo.CreateAppCallCount()).To(Equal(2))
			argTx, argSourceTerminalId, argAppGUID := egressPolicyRepo.CreateAppArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(argSourceTerminalId).To(Equal("some-term-guid"))
			Expect(argAppGUID).To(Equal("some-app-guid"))

			argTx, argSourceTerminalId, argAppGUID = egressPolicyRepo.CreateAppArgsForCall(1)
			Expect(argTx).To(Equal(tx))
			Expect(argSourceTerminalId).To(Equal("some-term-guid-2"))
			Expect(argAppGUID).To(Equal("different-app-guid"))

			Expect(egressPolicyRepo.CreateSpaceCallCount()).To(Equal(0))
		})

		It("returns an error when the CreateApp fails", func() {
			egressPolicyRepo.CreateAppReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).To(MatchError("failed to create source app: OMG WHY DID THIS FAIL"))
		})

		It("creates a space with a sourceTerminalGUID", func() {
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns("", nil)
			terminalsRepo.CreateReturns("some-term-guid", nil)
			_, err := egressPolicyStore.Create([]store.EgressPolicy{spacePolicy})
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateSpaceCallCount()).To(Equal(1))
			Expect(egressPolicyRepo.CreateAppCallCount()).To(Equal(0))
			argTx, argSourceTerminalGUID, argSpaceGUID := egressPolicyRepo.CreateSpaceArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(argSourceTerminalGUID).To(Equal("some-term-guid"))
			Expect(argSpaceGUID).To(Equal("space-guid"))
		})

		It("creates an egress policy with the right GUIDs", func() {
			terminalsRepo.CreateReturnsOnCall(0, "some-app-guid", nil)
			terminalsRepo.CreateReturnsOnCall(1, "some-space-guid", nil)

			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateEgressPolicyCallCount()).To(Equal(2))

			argTx, sourceID, destinationID := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal("some-app-guid"))
			Expect(destinationID).To(Equal("some-destination-guid"))

			argTx, sourceID, destinationID = egressPolicyRepo.CreateEgressPolicyArgsForCall(1)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal("some-space-guid"))
			Expect(destinationID).To(Equal("some-destination-guid-2"))
		})

		It("returns an error when the CreateEgressPolicy fails", func() {
			egressPolicyRepo.CreateEgressPolicyReturns("", errors.New("OMG WHY DID THIS FAIL"))

			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).To(MatchError("failed to create egress policy: OMG WHY DID THIS FAIL"))
		})

		It("uses the existing app terminal id when it exists", func() {
			egressPolicyRepo.GetTerminalByAppGUIDReturns("66", nil)

			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateAppCallCount()).To(Equal(0))
			_, sourceID, _ := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(sourceID).To(Equal("66"))
		})

		It("uses the existing space terminal id when it exists", func() {
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns("55", nil)

			_, err := egressPolicyStore.Create([]store.EgressPolicy{spacePolicy})
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateSpaceCallCount()).To(Equal(0))
			_, sourceID, _ := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(sourceID).To(Equal("55"))
		})

		It("returns an error when the CreateTerminal fails for space", func() {
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns("", nil)
			terminalsRepo.CreateReturns("", errors.New("OMG WHY DID THIS FAIL"))

			_, err := egressPolicyStore.Create([]store.EgressPolicy{spacePolicy})
			Expect(err).To(MatchError("failed to create source terminal: OMG WHY DID THIS FAIL"))
		})

		It("returns an error when the CreateSpace fails", func() {
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns("", nil)
			egressPolicyRepo.CreateSpaceReturns(-1, errors.New("OMG WHY DID THIS FAIL"))

			_, err := egressPolicyStore.Create([]store.EgressPolicy{spacePolicy})
			Expect(err).To(MatchError("failed to create space: OMG WHY DID THIS FAIL"))
		})

		It("returns an error when the GetTerminalBySpaceGUID fails", func() {
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns("", errors.New("OMG WHY DID THIS FAIL"))

			_, err := egressPolicyStore.Create([]store.EgressPolicy{spacePolicy})
			Expect(err).To(MatchError("failed to get terminal by space guid: OMG WHY DID THIS FAIL"))
		})

		It("returns an error when the GetTerminalByAppGUID fails", func() {
			egressPolicyRepo.GetTerminalByAppGUIDReturns("", errors.New("OMG WHY DID THIS FAIL"))

			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).To(MatchError("failed to get terminal by app guid: OMG WHY DID THIS FAIL"))
		})
	})

	Describe("Delete", func() {
		var (
			egressPoliciesToDelete    []store.EgressPolicy
			egressPolicyIDCollection  store.EgressPolicyIDCollection
			egressPolicyIDCollection2 store.EgressPolicyIDCollection

			egressPolicyGUID string
			ipRangeID        int64
			destTerminalGUID string
			appID            int64
			srcTerminalGUID  string

			egressPolicyGUID2 string
			ipRangeID2        int64
			destTerminalGUID2 string
			appID2            int64
			srcTerminalGUID2  string
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

			egressPolicyGUID = "some-egress-policy-guid"
			ipRangeID = 9
			destTerminalGUID = "some-dest-terminal-guid"
			appID = 21
			srcTerminalGUID = "some-src-terminal-guid"

			egressPolicyGUID2 = "some-egress-policy-guid-2"
			ipRangeID2 = 10
			destTerminalGUID2 = "some-dest-terminal-guid-2"
			appID2 = 23
			srcTerminalGUID2 = "some-src-terminal-guid-2"

			egressPolicyIDCollection = store.EgressPolicyIDCollection{
				EgressPolicyGUID:        egressPolicyGUID,
				DestinationIPRangeID:    ipRangeID,
				DestinationTerminalGUID: destTerminalGUID,
				SourceAppID:             appID,
				SourceSpaceID:           -1,
				SourceTerminalGUID:      srcTerminalGUID,
			}

			egressPolicyIDCollection2 = store.EgressPolicyIDCollection{
				EgressPolicyGUID:        egressPolicyGUID2,
				DestinationIPRangeID:    ipRangeID2,
				DestinationTerminalGUID: destTerminalGUID2,
				SourceAppID:             appID2,
				SourceSpaceID:           -1,
				SourceTerminalGUID:      srcTerminalGUID2,
			}

			egressPolicyIDCollections := []store.EgressPolicyIDCollection{
				egressPolicyIDCollection,
				egressPolicyIDCollection2,
			}
			egressPolicyRepo.GetIDCollectionsByEgressPolicyReturns(egressPolicyIDCollections, nil)
		})

		It("returns an error when beginning a transaction fails", func() {
			mockDb.BeginxReturns(nil, errors.New("failed to create tx"))
			err := egressPolicyStore.Delete(egressPoliciesToDelete)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("create transaction: failed to create tx"))
		})

		It("deletes the egress policy", func() {
			err := egressPolicyStore.Delete(egressPoliciesToDelete)
			Expect(err).NotTo(HaveOccurred())

			Expect(egressPolicyRepo.GetIDCollectionsByEgressPolicyCallCount()).To(Equal(1))
			passedTx, passedEgressPolicy := egressPolicyRepo.GetIDCollectionsByEgressPolicyArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedEgressPolicy).To(Equal(egressPoliciesToDelete[0]))

			Expect(egressPolicyRepo.DeleteEgressPolicyCallCount()).To(Equal(2))
			passedTx, passedEgressPolicyGUID := egressPolicyRepo.DeleteEgressPolicyArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedEgressPolicyGUID).To(Equal(egressPolicyGUID))
			passedTx, passedEgressPolicyGUID = egressPolicyRepo.DeleteEgressPolicyArgsForCall(1)
			Expect(passedTx).To(Equal(tx))
			Expect(passedEgressPolicyGUID).To(Equal(egressPolicyGUID2))

			Expect(egressPolicyRepo.DeleteIPRangeCallCount()).To(Equal(2))
			passedTx, passedIPRangeID := egressPolicyRepo.DeleteIPRangeArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedIPRangeID).To(Equal(ipRangeID))
			passedTx, passedIPRangeID = egressPolicyRepo.DeleteIPRangeArgsForCall(1)
			Expect(passedTx).To(Equal(tx))
			Expect(passedIPRangeID).To(Equal(ipRangeID2))

			Expect(terminalsRepo.DeleteCallCount()).To(Equal(4))
			passedTx, passedDestTerminalGUID := terminalsRepo.DeleteArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedDestTerminalGUID).To(Equal(destTerminalGUID))

			passedTx, passedSrcTerminalGUID := terminalsRepo.DeleteArgsForCall(1)
			Expect(passedTx).To(Equal(tx))
			Expect(passedSrcTerminalGUID).To(Equal(srcTerminalGUID))

			passedTx, passedDestTerminalGUID = terminalsRepo.DeleteArgsForCall(2)
			Expect(passedTx).To(Equal(tx))
			Expect(passedDestTerminalGUID).To(Equal(destTerminalGUID2))

			passedTx, passedSrcTerminalGUID = terminalsRepo.DeleteArgsForCall(3)
			Expect(passedTx).To(Equal(tx))
			Expect(passedSrcTerminalGUID).To(Equal(srcTerminalGUID2))

			Expect(egressPolicyRepo.DeleteAppCallCount()).To(Equal(2))
			passedTx, passedAppID := egressPolicyRepo.DeleteAppArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedAppID).To(Equal(appID))

			passedTx, passedAppID = egressPolicyRepo.DeleteAppArgsForCall(1)
			Expect(passedTx).To(Equal(tx))
			Expect(passedAppID).To(Equal(appID2))

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
				egressPolicyIDCollections := []store.EgressPolicyIDCollection{egressPolicyIDCollection}
				egressPolicyRepo.GetIDCollectionsByEgressPolicyReturns(egressPolicyIDCollections, nil)
			})

			It("deletes the space", func() {
				err := egressPolicyStore.Delete(egressPoliciesToDelete)
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
					err := egressPolicyStore.Delete(egressPoliciesToDelete)
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
				err := egressPolicyStore.Delete(egressPoliciesToDelete)
				Expect(err).NotTo(HaveOccurred())

				Expect(egressPolicyRepo.GetIDCollectionsByEgressPolicyCallCount()).To(Equal(2))
				passedTx, passedEgressPolicy := egressPolicyRepo.GetIDCollectionsByEgressPolicyArgsForCall(0)
				Expect(passedTx).To(Equal(tx))
				Expect(passedEgressPolicy).To(Equal(egressPoliciesToDelete[0]))

				passedTx, passedEgressPolicy = egressPolicyRepo.GetIDCollectionsByEgressPolicyArgsForCall(1)
				Expect(passedTx).To(Equal(tx))
				Expect(passedEgressPolicy).To(Equal(egressPoliciesToDelete[1]))

				Expect(egressPolicyRepo.DeleteEgressPolicyCallCount()).To(Equal(4))
				Expect(egressPolicyRepo.DeleteIPRangeCallCount()).To(Equal(4))
				Expect(terminalsRepo.DeleteCallCount()).To(Equal(8))
				Expect(egressPolicyRepo.DeleteAppCallCount()).To(Equal(4))
			})
		})

		Context("when app is referenced by another egress policy", func() {
			BeforeEach(func() {
				egressPolicyRepo.IsTerminalInUseReturns(true, nil)
			})

			It("doesn't delete the source terminal or source app", func() {
				err := egressPolicyStore.Delete(egressPoliciesToDelete)
				Expect(err).NotTo(HaveOccurred())

				Expect(egressPolicyRepo.DeleteAppCallCount()).To(Equal(0))
			})
		})

		Context("when the deleteWithTx fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetIDCollectionsByEgressPolicyReturns([]store.EgressPolicyIDCollection{}, errors.New("ther's a bug"))
			})

			It("rollsback the transaction", func() {
				err := egressPolicyStore.Delete(egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to find egress policy: ther's a bug"))
				Expect(tx.RollbackCallCount()).To(Equal(1))
			})
		})

		Context("when the EgressPolicyRepo.GetIDCollectionsByEgressPolicy fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetIDCollectionsByEgressPolicyReturns([]store.EgressPolicyIDCollection{}, errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.Delete(egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to find egress policy: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteEgressPolicy fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.DeleteEgressPolicyReturns(errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.Delete(egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to delete egress policy: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteIPRange fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.DeleteIPRangeReturns(errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.Delete(egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to delete destination ip range: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteTerminal fails", func() {
			BeforeEach(func() {
				terminalsRepo.DeleteReturns(errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.Delete(egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to delete destination terminal: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.IsTerminalInUse fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.IsTerminalInUseReturns(false, errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.Delete(egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to check if source terminal is in use: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteApp fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.DeleteAppReturns(errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.Delete(egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to delete source app: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteTerminal fails", func() {
			BeforeEach(func() {
				terminalsRepo.DeleteReturnsOnCall(1, errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				err := egressPolicyStore.Delete(egressPoliciesToDelete)
				Expect(err).To(MatchError("failed to delete source terminal: ther's a bug"))
			})
		})

		It("returns an error when commit transaction fails", func() {
			tx.CommitReturns(errors.New("failed to commit"))
			err := egressPolicyStore.Delete(egressPoliciesToDelete)
			Expect(err).To(MatchError("commit transaction: failed to commit"))
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

	Describe("GetBySourceGuids", func() {
		Context("when called with ids", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetBySourceGuidsReturns(egressPolicies, nil)
			})

			It("calls egressPolicyRepo.GetByGuid", func() {
				policies, err := egressPolicyStore.GetBySourceGuids([]string{"meow"})
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(Equal(egressPolicies))

				ids := egressPolicyRepo.GetBySourceGuidsArgsForCall(0)
				Expect(ids).To(Equal([]string{"meow"}))
			})
		})

		Context("when an error is returned from the repo", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetBySourceGuidsReturns(nil, errors.New("bark bark"))
			})

			It("calls egressPolicyRepo.GetByGuid", func() {
				_, err := egressPolicyStore.GetBySourceGuids([]string{"meow"})
				Expect(err).To(MatchError("failed to get policies by guids: bark bark"))
			})
		})
	})
})
