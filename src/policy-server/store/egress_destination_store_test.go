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
	uuid "github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressDestinationStore", func() {
	var (
		egressDestinationsStore *store.EgressDestinationStore
		destinationMetadataRepo *store.DestinationMetadataTable
		terminalsRepo           *store.TerminalsTable
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
				TerminalsRepo:           terminalsRepo,
				DestinationMetadataRepo: destinationMetadataRepo,
				Conn: realDb,
				EgressDestinationRepo: egressDestinationTable,
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

				_, err = uuid.ParseHex(createdDestinations[0].GUID)
				Expect(err).NotTo(HaveOccurred())
				Expect(createdDestinations[0].Name).To(Equal("dest-1"))
				Expect(createdDestinations[0].Description).To(Equal("desc-1"))
				Expect(createdDestinations[0].Protocol).To(Equal("tcp"))
				Expect(createdDestinations[0].IPRanges).To(Equal([]store.IPRange{{Start: "1.2.2.2", End: "1.2.2.3"}}))
				Expect(createdDestinations[0].Ports).To(Equal([]store.Ports{{Start: 8080, End: 8081}}))

				_, err = uuid.ParseHex(createdDestinations[1].GUID)
				Expect(err).NotTo(HaveOccurred())
				Expect(createdDestinations[1].Name).To(Equal("dest-2"))
				Expect(createdDestinations[1].Description).To(Equal("desc-2"))
				Expect(createdDestinations[1].Protocol).To(Equal("icmp"))
				Expect(createdDestinations[1].IPRanges).To(Equal([]store.IPRange{{Start: "1.2.2.4", End: "1.2.2.5"}}))
				Expect(createdDestinations[1].ICMPType).To(Equal(12))
				Expect(createdDestinations[1].ICMPCode).To(Equal(13))

				destinations, err := egressDestinationsStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations[0].GUID).To(Equal(createdDestinations[0].GUID))
				Expect(destinations[0].Name).To(Equal("dest-1"))
				Expect(destinations[0].Description).To(Equal("desc-1"))
				Expect(destinations[0].Protocol).To(Equal("tcp"))
				Expect(destinations[0].IPRanges).To(Equal([]store.IPRange{{Start: "1.2.2.2", End: "1.2.2.3"}}))
				Expect(destinations[0].Ports).To(Equal([]store.Ports{{Start: 8080, End: 8081}}))

				Expect(destinations[1].GUID).To(Equal(createdDestinations[1].GUID))
				Expect(destinations[1].Name).To(Equal("dest-2"))
				Expect(destinations[1].Description).To(Equal("desc-2"))
				Expect(destinations[1].Protocol).To(Equal("icmp"))
				Expect(destinations[1].IPRanges).To(Equal([]store.IPRange{{Start: "1.2.2.4", End: "1.2.2.5"}}))
				Expect(destinations[1].ICMPType).To(Equal(12))
				Expect(destinations[1].ICMPCode).To(Equal(13))
			})
		})
	})

	Context("db error cases using mock", func() {
		var (
			mockDB                  *fakes.Db
			tx                      *dbfakes.Transaction
			terminalsRepo           *fakes.TerminalsRepo
			egressDestinationRepo   *fakes.EgressDestinationRepo
			destinationMetadataRepo *fakes.DestinationMetadataRepo
		)

		BeforeEach(func() {
			mockDB = new(fakes.Db)
			tx = new(dbfakes.Transaction)

			mockDB.BeginxReturns(tx, nil)

			terminalsRepo = &fakes.TerminalsRepo{}
			egressDestinationRepo = &fakes.EgressDestinationRepo{}
			destinationMetadataRepo = &fakes.DestinationMetadataRepo{}

			egressDestinationsStore = &store.EgressDestinationStore{
				Conn: mockDB,
				EgressDestinationRepo:   egressDestinationRepo,
				DestinationMetadataRepo: destinationMetadataRepo,
				TerminalsRepo:           terminalsRepo,
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
					terminalsRepo.CreateReturns("", errors.New("can't create a terminal"))
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
