package store_test

import (
	"errors"
	"fmt"
	"policy-server/db"
	dbfakes "policy-server/db/fakes"
	"policy-server/store"
	"policy-server/store/fakes"
	"test-helpers"
	"time"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressDestinationStore", func() {
	var (
		egressDestinationsStore *store.EgressDestinationStore
		egressPolicyRepo        *store.EgressPolicyTable
		destinationMetadataRepo *store.DestinationMetadataTable
		egressDestinationTable  *store.EgressDestinationTable
	)

	Describe("using an actual db", func() {
		var (
			dbConf dbHelper.Config
			realDb *db.ConnWrapper
		)

		BeforeEach(func() {
			dbConf = testsupport.GetDBConfig()
			dbConf.DatabaseName = fmt.Sprintf("egress_destination_store_test_node_%d", time.Now().UnixNano())
			dbConf.Timeout = 30
			testhelpers.CreateDatabase(dbConf)

			logger := lager.NewLogger("Egress Destination Store Test")
			realDb = db.NewConnectionPool(dbConf, 200, 200, "Egress Destination Store Test", "Egress Destination Store Test", logger)

			migrate(realDb)

			egressDestinationTable = &store.EgressDestinationTable{}
			destinationMetadataRepo = &store.DestinationMetadataTable{}

			egressDestinationsStore = &store.EgressDestinationStore{
				Conn: realDb,
				EgressDestinationRepo:   egressDestinationTable,
				TerminalRepo:            egressPolicyRepo,
				DestinationMetadataRepo: destinationMetadataRepo,
			}
			egressPolicyRepo = &store.EgressPolicyTable{
				Conn: realDb,
			}
		})

		AfterEach(func() {
			if realDb != nil {
				Expect(realDb.Close()).To(Succeed())
			}
			testhelpers.RemoveDatabase(dbConf)
		})

		Describe("CRUD", func() {
			var (
				toBeCreatedDestinations []store.EgressDestination
			)

			BeforeEach(func() {
				toBeCreatedDestinations = []store.EgressDestination{
					{
						Name:        "dest-1",
						Description: "desc-1",
						Protocol:    "tcp",
						IPRanges:    []store.IPRange{{Start: "1.2.2.2", End: "1.2.2.3"}},
						Ports:       []store.Ports{{Start: 8080, End: 8081}},
					},
					{
						Name:        "dest-2",
						Description: "desc-2",
						Protocol:    "icmp",
						IPRanges:    []store.IPRange{{Start: "1.2.2.4", End: "1.2.2.5"}},
						ICMPType:    12,
						ICMPCode:    13,
					},
				}
			})

			It("creates and lists policies to/from the database", func() {
				createdDestinations, err := egressDestinationsStore.Create(toBeCreatedDestinations)
				Expect(err).NotTo(HaveOccurred())
				Expect(createdDestinations).To(HaveLen(2))
				Expect(createdDestinations).To(Equal([]store.EgressDestination{
					{
						ID:          "1",
						Name:        "dest-1",
						Description: "desc-1",
						Protocol:    "tcp",
						IPRanges:    []store.IPRange{{Start: "1.2.2.2", End: "1.2.2.3"}},
						Ports:       []store.Ports{{Start: 8080, End: 8081}},
					},
					{
						ID:          "2",
						Name:        "dest-2",
						Description: "desc-2",
						Protocol:    "icmp",
						IPRanges:    []store.IPRange{{Start: "1.2.2.4", End: "1.2.2.5"}},
						ICMPType:    12,
						ICMPCode:    13,
					},
				}))

				destinations, err := egressDestinationsStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(Equal([]store.EgressDestination{
					{
						ID:          "1",
						Name:        "dest-1",
						Description: "desc-1",
						Protocol:    "tcp",
						IPRanges:    []store.IPRange{{Start: "1.2.2.2", End: "1.2.2.3"}},
						Ports:       []store.Ports{{Start: 8080, End: 8081}},
					},
					{
						ID:          "2",
						Name:        "dest-2",
						Description: "desc-2",
						Protocol:    "icmp",
						IPRanges:    []store.IPRange{{Start: "1.2.2.4", End: "1.2.2.5"}},
						ICMPType:    12,
						ICMPCode:    13,
					},
				}))
			})
		})
	})

	Context("db error cases using mock", func() {
		var (
			mockDB                  *fakes.Db
			tx                      *dbfakes.Transaction
			terminalRepo            *fakes.TerminalRepo
			egressDestinationRepo   *fakes.EgressDestinationRepo
			destinationMetadataRepo *fakes.DestinationMetadataRepo
		)

		BeforeEach(func() {
			mockDB = new(fakes.Db)
			tx = new(dbfakes.Transaction)

			mockDB.BeginxReturns(tx, nil)

			terminalRepo = &fakes.TerminalRepo{}
			egressDestinationRepo = &fakes.EgressDestinationRepo{}
			destinationMetadataRepo = &fakes.DestinationMetadataRepo{}

			egressDestinationsStore = &store.EgressDestinationStore{
				Conn: mockDB,
				EgressDestinationRepo:   egressDestinationRepo,
				DestinationMetadataRepo: destinationMetadataRepo,
				TerminalRepo:            terminalRepo,
			}
		})

		Context("Create", func() {
			Context("when the transaction cannot be created", func() {
				BeforeEach(func() {
					mockDB.BeginxReturns(nil, errors.New("can't create a transaction"))
				})

				It("returns an error", func() {
					_, err := egressDestinationsStore.Create([]store.EgressDestination{})
					Expect(err).To(MatchError("egress destination store create transaction: can't create a transaction"))
				})
			})

			Context("when creating the terminal returns an error", func() {
				BeforeEach(func() {
					terminalRepo.CreateTerminalReturns(-1, errors.New("can't create a terminal"))
				})

				It("returns an error", func() {
					_, err := egressDestinationsStore.Create([]store.EgressDestination{{}})
					Expect(err).To(MatchError("egress destination store create terminal: can't create a terminal"))
				})

				It("rolls back the transaction", func() {
					egressDestinationsStore.Create([]store.EgressDestination{{}})
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("when creating the destination metadata returns an error", func() {
				var err error

				BeforeEach(func() {
					destinationMetadataRepo.CreateReturns(-1, errors.New("can't create a destination metadata"))
					_, err = egressDestinationsStore.Create([]store.EgressDestination{
						{
							Name:        " ",
							Description: " ",
							Protocol:    "icmp",
							IPRanges:    []store.IPRange{{Start: "2.2.2.4", End: "2.2.2.5"}},
							ICMPType:    11,
							ICMPCode:    14,
						},
					})
				})

				It("returns an error", func() {
					Expect(err).To(MatchError("egress destination store create destination metadata: can't create a destination metadata"))
				})

				It("rolls back the transaction", func() {
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("when creating the ip range returns an error", func() {
				var err error
				BeforeEach(func() {
					egressDestinationRepo.CreateIPRangeReturns(-1, errors.New("can't create an ip range"))
					_, err = egressDestinationsStore.Create([]store.EgressDestination{
						{
							Name:        " ",
							Description: " ",
							Protocol:    "icmp",
							IPRanges:    []store.IPRange{{Start: "2.2.2.4", End: "2.2.2.5"}},
							ICMPType:    11,
							ICMPCode:    14,
						},
					})
				})

				It("returns an error", func() {
					Expect(err).To(MatchError("egress destination store create ip range: can't create an ip range"))
				})

				It("rolls back the transaction", func() {
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("when the transaction cannot be committed", func() {
				var err error
				BeforeEach(func() {
					tx.CommitReturns(errors.New("can't commit transaction"))
					_, err = egressDestinationsStore.Create([]store.EgressDestination{})
				})

				It("returns an error", func() {
					Expect(err).To(MatchError("egress destination store commit transaction: can't commit transaction"))
				})

				It("rolls back the transaction", func() {
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})
		})

		Context("All", func() {
			Context("when the transaction cannot be created", func() {
				BeforeEach(func() {
					mockDB.BeginxReturns(nil, errors.New("can't create a transaction"))
				})

				It("returns an error", func() {
					_, err := egressDestinationsStore.All()
					Expect(err).To(MatchError("egress destination store create transaction: can't create a transaction"))
				})
			})
		})
	})
})
