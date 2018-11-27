package store_test

import (
	"errors"
	"policy-server/store"
	"policy-server/store/fakes"

	dbfakes "code.cloudfoundry.org/cf-networking-helpers/db/fakes"
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
				AppLifecycle: "running",
			},
			{
				Source: store.EgressSource{
					ID: "different-app-guid",
				},
				Destination: store.EgressDestination{
					GUID: "some-destination-guid-2",
				},
				AppLifecycle: "staging",
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
			AppLifecycle: "all",
		}

		egressPolicyRepo.GetTerminalByAppGUIDReturns("", nil)
	})

	Describe("Create", func() {
		It("creates an egress policy with the right GUIDs", func() {
			terminalsRepo.CreateReturnsOnCall(0, "some-terminal-app-guid", nil)
			terminalsRepo.CreateReturnsOnCall(1, "some-terminal-space-guid", nil)
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
						ID:           "some-app-guid",
						TerminalGUID: "some-terminal-app-guid",
					},
					Destination: store.EgressDestination{
						GUID: "some-destination-guid",
					},
					AppLifecycle: "running",
				},
				{
					ID: "some-egress-policy-guid-2",
					Source: store.EgressSource{
						ID:           "different-app-guid",
						TerminalGUID: "some-terminal-space-guid",
					},
					Destination: store.EgressDestination{
						GUID: "some-destination-guid-2",
					},
					AppLifecycle: "staging",
				},
			}))

			argTx, sourceID, destinationID, appLifecycle := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal("some-terminal-app-guid"))
			Expect(destinationID).To(Equal("some-destination-guid"))
			Expect(appLifecycle).To(Equal("running"))

			argTx, sourceID, destinationID, appLifecycle = egressPolicyRepo.CreateEgressPolicyArgsForCall(1)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal("some-terminal-space-guid"))
			Expect(destinationID).To(Equal("some-destination-guid-2"))
			Expect(appLifecycle).To(Equal("staging"))
		})

		It("returns an error when get by filter fails", func() {
			egressPolicyRepo.GetByFilterReturns(nil, errors.New("bark bark"))
			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).To(MatchError("failed to filter existing policies: failed to get policies by filter: bark bark"))
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
			egressPolicyRepo.CreateEgressPolicyReturns("some-egress-policy-guid", nil)
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns("", nil)
			terminalsRepo.CreateReturns("some-term-guid", nil)
			createdPolicies, err := egressPolicyStore.Create([]store.EgressPolicy{spacePolicy})
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateSpaceCallCount()).To(Equal(1))
			Expect(egressPolicyRepo.CreateAppCallCount()).To(Equal(0))
			argTx, argSourceTerminalGUID, argSpaceGUID := egressPolicyRepo.CreateSpaceArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(argSourceTerminalGUID).To(Equal("some-term-guid"))
			Expect(argSpaceGUID).To(Equal("space-guid"))

			Expect(createdPolicies).To(Equal([]store.EgressPolicy{{
				ID: "some-egress-policy-guid",
				Source: store.EgressSource{
					Type: "space",
					ID:   "space-guid",
					TerminalGUID: "some-term-guid",
				},
				Destination: store.EgressDestination{
					GUID: "some-destination-guid",
				},
				AppLifecycle: "all",
			}}))
		})

		It("creates an egress policy with the right GUIDs", func() {
			terminalsRepo.CreateReturnsOnCall(0, "some-app-guid", nil)
			terminalsRepo.CreateReturnsOnCall(1, "some-space-guid", nil)

			_, err := egressPolicyStore.Create(egressPolicies)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateEgressPolicyCallCount()).To(Equal(2))

			argTx, sourceID, destinationID, appLifecycle := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal("some-app-guid"))
			Expect(destinationID).To(Equal("some-destination-guid"))
			Expect(appLifecycle).To(Equal("running"))

			argTx, sourceID, destinationID, appLifecycle = egressPolicyRepo.CreateEgressPolicyArgsForCall(1)
			Expect(argTx).To(Equal(tx))
			Expect(sourceID).To(Equal("some-space-guid"))
			Expect(destinationID).To(Equal("some-destination-guid-2"))
			Expect(appLifecycle).To(Equal("staging"))
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
			_, sourceID, _, _ := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
			Expect(sourceID).To(Equal("66"))
		})

		It("uses the existing space terminal id when it exists", func() {
			egressPolicyRepo.GetTerminalBySpaceGUIDReturns("55", nil)

			_, err := egressPolicyStore.Create([]store.EgressPolicy{spacePolicy})
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicyRepo.CreateSpaceCallCount()).To(Equal(0))
			_, sourceID, _, _ := egressPolicyRepo.CreateEgressPolicyArgsForCall(0)
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
			egressPolicyGUID string
			destTerminalGUID string
			srcTerminalGUID  string
			srcGUID          string
			srcType          string

			egressPolicyGUID2 string
			destTerminalGUID2 string
			srcTerminalGUID2  string
			srcGUID2          string
			srcType2          string

			expectedEgressPolicies []store.EgressPolicy
		)
		BeforeEach(func() {
			egressPolicyGUID = "some-egress-policy-guid"
			destTerminalGUID = "some-dest-terminal-guid"
			srcTerminalGUID = "some-src-terminal-guid"
			srcGUID = "some-app-guid"
			srcType = "app"

			egressPolicyGUID2 = "some-egress-policy-guid-2"
			destTerminalGUID2 = "some-dest-terminal-guid-2"
			srcTerminalGUID2 = "some-src-terminal-guid-2"
			srcGUID2 = "some-space-guid"
			srcType2 = "space"

			expectedEgressPolicies = []store.EgressPolicy{
				{
					ID: egressPolicyGUID,
					Source: store.EgressSource{
						TerminalGUID: srcTerminalGUID,
						ID:           srcGUID,
						Type:         srcType,
					},
					Destination: store.EgressDestination{
						GUID: destTerminalGUID,
					},
					AppLifecycle: "running",
				},
				{
					ID: egressPolicyGUID2,
					Source: store.EgressSource{
						TerminalGUID: srcTerminalGUID2,
						ID:           srcGUID2,
						Type:         srcType2,
					},
					Destination: store.EgressDestination{
						GUID: destTerminalGUID2,
					},
					AppLifecycle: "staging",
				},
			}
			egressPolicyRepo.GetByGUIDReturns(expectedEgressPolicies, nil)
		})

		It("returns an error when beginning a transaction fails", func() {
			mockDb.BeginxReturns(nil, errors.New("failed to create tx"))
			_, err := egressPolicyStore.Delete(egressPolicyGUID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("create transaction: failed to create tx"))
		})

		It("deletes the egress policies and returns the deleted egress policies", func() {
			egressPolicies, err := egressPolicyStore.Delete(egressPolicyGUID, egressPolicyGUID2)
			Expect(err).NotTo(HaveOccurred())
			Expect(egressPolicies).To(Equal(expectedEgressPolicies))

			Expect(egressPolicyRepo.GetByGUIDCallCount()).To(Equal(1))
			passedTx, passedEgressPolicyGUIDs := egressPolicyRepo.GetByGUIDArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedEgressPolicyGUIDs).To(ConsistOf(egressPolicyGUID, egressPolicyGUID2))

			Expect(egressPolicyRepo.DeleteEgressPolicyCallCount()).To(Equal(2))
			passedTx, passedEgressPolicyGUID := egressPolicyRepo.DeleteEgressPolicyArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedEgressPolicyGUID).To(Equal(egressPolicyGUID))

			passedTx, passedEgressPolicyGUID = egressPolicyRepo.DeleteEgressPolicyArgsForCall(1)
			Expect(passedTx).To(Equal(tx))
			Expect(passedEgressPolicyGUID).To(Equal(egressPolicyGUID2))

			Expect(terminalsRepo.DeleteCallCount()).To(Equal(2))
			passedTx, passedSrcTerminalGUID := terminalsRepo.DeleteArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedSrcTerminalGUID).To(Equal(srcTerminalGUID))

			passedTx, passedSrcTerminalGUID = terminalsRepo.DeleteArgsForCall(1)
			Expect(passedTx).To(Equal(tx))
			Expect(passedSrcTerminalGUID).To(Equal(srcTerminalGUID2))

			Expect(egressPolicyRepo.DeleteAppCallCount()).To(Equal(1))
			passedTx, passedSrcTerminalGUID = egressPolicyRepo.DeleteAppArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedSrcTerminalGUID).To(Equal(srcTerminalGUID))

			Expect(egressPolicyRepo.DeleteSpaceCallCount()).To(Equal(1))
			passedTx, passedSourceTerminalGUID := egressPolicyRepo.DeleteSpaceArgsForCall(0)
			Expect(passedTx).To(Equal(tx))
			Expect(passedSourceTerminalGUID).To(Equal(srcTerminalGUID2))
		})

		Context("when the EgressPolicyRepo.DeleteSpace fails", func() {
			BeforeEach(func() {
				srcGUID = "some-space-guid"
				srcType = "space"
				expectedEgressPolicies = []store.EgressPolicy{
					{
						ID: egressPolicyGUID,
						Source: store.EgressSource{
							TerminalGUID: srcTerminalGUID,
							ID:           srcGUID,
							Type:         srcType,
						},
						Destination: store.EgressDestination{
							GUID: destTerminalGUID,
						},
						AppLifecycle: "all",
					},
				}
				egressPolicyRepo.GetByGUIDReturns(expectedEgressPolicies, nil)
				egressPolicyRepo.DeleteSpaceReturns(errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				_, err := egressPolicyStore.Delete(egressPolicyGUID)
				Expect(err).To(MatchError("failed to delete source space: ther's a bug"))
			})
		})

		Context("when the egress policy doesn't exist", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetByGUIDReturns([]store.EgressPolicy{}, nil)
			})

			It("returns an empty array", func() {
				egressPolicies, err := egressPolicyStore.Delete(egressPolicyGUID)
				Expect(err).NotTo(HaveOccurred())
				Expect(egressPolicies).To(HaveLen(0))
			})
		})

		Context("when app is referenced by another egress policy", func() {
			BeforeEach(func() {
				egressPolicyRepo.IsTerminalInUseReturns(true, nil)
			})

			It("doesn't delete the source terminal or source app", func() {
				_, err := egressPolicyStore.Delete(egressPolicyGUID)
				Expect(err).NotTo(HaveOccurred())

				Expect(egressPolicyRepo.DeleteAppCallCount()).To(Equal(0))
			})
		})

		Context("when the deleteWithTx fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetByGUIDReturns(nil, errors.New("ther's a bug"))
			})

			It("rollsback the transaction", func() {
				_, err := egressPolicyStore.Delete(egressPolicyGUID)
				Expect(err).To(MatchError("failed to find egress policy: ther's a bug"))
				Expect(tx.RollbackCallCount()).To(Equal(1))
			})
		})

		Context("when the EgressPolicyRepo.GetByGUID fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetByGUIDReturns(nil, errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				_, err := egressPolicyStore.Delete(egressPolicyGUID)
				Expect(err).To(MatchError("failed to find egress policy: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteEgressPolicy fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.DeleteEgressPolicyReturns(errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				_, err := egressPolicyStore.Delete(egressPolicyGUID)
				Expect(err).To(MatchError("failed to delete egress policy: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.IsTerminalInUse fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.IsTerminalInUseReturns(false, errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				_, err := egressPolicyStore.Delete(egressPolicyGUID)
				Expect(err).To(MatchError("failed to check if source terminal is in use: ther's a bug"))
			})
		})

		Context("when the EgressPolicyRepo.DeleteApp fails", func() {
			BeforeEach(func() {
				egressPolicyRepo.DeleteAppReturns(errors.New("ther's a bug"))
			})

			It("returns an error", func() {
				_, err := egressPolicyStore.Delete(egressPolicyGUID)
				Expect(err).To(MatchError("failed to delete source app: ther's a bug"))
			})
		})

		It("returns an error when commit transaction fails", func() {
			tx.CommitReturns(errors.New("failed to commit"))
			_, err := egressPolicyStore.Delete(egressPolicyGUID)
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

	Describe("GetByFilter", func() {
		Context("when called", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetByFilterReturns(egressPolicies, nil)
			})

			It("calls egressPolicyRepo.GetByFilter", func() {
				policies, err := egressPolicyStore.GetByFilter([]string{"abc-123"}, []string{"outerSpace"}, []string{"xyz789"}, []string{"moon walk"})
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(Equal(egressPolicies))

				sourceIds, sourceTypes, destinationIds, destinationNames := egressPolicyRepo.GetByFilterArgsForCall(0)
				Expect(sourceIds).To(Equal([]string{"abc-123"}))
				Expect(sourceTypes).To(Equal([]string{"outerSpace"}))
				Expect(destinationIds).To(Equal([]string{"xyz789"}))
				Expect(destinationNames).To(Equal([]string{"moon walk"}))
			})
		})

		Context("when an error is returned from the repo", func() {
			BeforeEach(func() {
				egressPolicyRepo.GetByFilterReturns(nil, errors.New("bark bark"))
			})

			It("calls egressPolicyRepo.GetByFilter", func() {
				_, err := egressPolicyStore.GetByFilter([]string{"sourceId"}, []string{"sourceType"}, []string{"destinationId"}, []string{"destinationName"})
				Expect(err).To(MatchError("failed to get policies by filter: bark bark"))
			})
		})
	})
})
