package store_test

import (
	"errors"
	"fmt"
	"policy-server/store"
	"policy-server/store/fakes"
	"test-helpers"
	"time"

	dbfakes "code.cloudfoundry.org/cf-networking-helpers/db/fakes"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"
	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func dumpQueries(realDb *dbHelper.ConnWrapper) {
	fmt.Println("")
	fmt.Println("====================== Begin query dump =======================")
	rows, err := realDb.Query(`
				SELECT ID, USER, HOST, DB, COMMAND, STATE, INFO
				FROM INFORMATION_SCHEMA.PROCESSLIST
			`)
	Expect(err).ToNot(HaveOccurred())

	for rows.Next() {
		var id, user, host, db, command, state, info *string
		err = rows.Scan(&id, &user, &host, &db, &command, &state, &info)
		if err != nil {
			rows.Close()
			fmt.Printf("error reading row: %s", err)
		}

		if info == nil {
			var none = "<null>"
			info = &none
		}
		fmt.Printf("%s, %s, %s, %s, %s, %s, %s\n", *id, *user, *host, *db, *command, *state, *info)
	}
	rows.Close()
	fmt.Println("====================== End query dump =======================")
	fmt.Println("")
}

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
			realDb *dbHelper.ConnWrapper
		)

		BeforeEach(func() {
			dbConf = testsupport.GetDBConfig()
			dbConf.DatabaseName = fmt.Sprintf("egress_destination_store_test_node_%d", time.Now().UnixNano())
			dbConf.Timeout = 30
			testhelpers.CreateDatabase(dbConf)

			logger := lager.NewLogger("Egress Destination Store Test")

			var err error
			realDb, err = dbHelper.NewConnectionPool(dbConf, 200, 200, 5*time.Minute, "Egress Destination Store Test", "Egress Destination Store Test", logger)
			Expect(err).NotTo(HaveOccurred())

			migrate(realDb)

			egressDestinationTable = &store.EgressDestinationTable{}
			destinationMetadataRepo = &store.DestinationMetadataTable{}
			terminalsRepo = &store.TerminalsTable{
				Guids: &store.GuidGenerator{},
			}

			egressDestinationsStore = &store.EgressDestinationStore{
				TerminalsRepo:           terminalsRepo,
				DestinationMetadataRepo: destinationMetadataRepo,
				Conn: realDb,
				EgressDestinationRepo: egressDestinationTable,
			}
		})

		AfterEach(func() {
			Expect(realDb.Close()).To(Succeed())
			testhelpers.RemoveDatabase(dbConf)
		})

		Describe("CRUD", func() {
			var (
				toBeCreatedDestinations []store.EgressDestination
				egressPolicyStore       *store.EgressPolicyStore
				egressPolicyRepo        *store.EgressPolicyTable

				createdDestinations []store.EgressDestination
			)

			BeforeEach(func() {
				egressPolicyRepo = &store.EgressPolicyTable{
					Conn:  realDb,
					Guids: &store.GuidGenerator{},
				}
				egressPolicyStore = &store.EgressPolicyStore{
					TerminalsRepo:    terminalsRepo,
					EgressPolicyRepo: egressPolicyRepo,
					Conn:             realDb,
				}

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

			It("creates, lists, and deletes destinations to/from the database", func() {
				By("creating")
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

				By("listing")
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
				Expect(destinations[1].Ports).To(HaveLen(0))
				Expect(destinations[1].ICMPType).To(Equal(12))
				Expect(destinations[1].ICMPCode).To(Equal(13))

				By("getting")
				destinations, err = egressDestinationsStore.GetByGUID(createdDestinations[0].GUID)
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations[0].GUID).To(Equal(createdDestinations[0].GUID))
				Expect(destinations[0].Name).To(Equal("dest-1"))
				Expect(destinations[0].Description).To(Equal("desc-1"))
				Expect(destinations[0].Protocol).To(Equal("tcp"))
				Expect(destinations[0].IPRanges).To(Equal([]store.IPRange{{Start: "1.2.2.2", End: "1.2.2.3"}}))
				Expect(destinations[0].Ports).To(Equal([]store.Ports{{Start: 8080, End: 8081}}))

				destinations, err = egressDestinationsStore.GetByGUID("unknown-guid")
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(HaveLen(0))

				By("getting by name")
				destinations, err = egressDestinationsStore.GetByName("dest-1", "dest-2")
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(HaveLen(2))
				Expect(destinations[0].Name).To(Equal("dest-1"))
				Expect(destinations[1].Name).To(Equal("dest-2"))

				By("getting by nonexistant name")
				destinations, err = egressDestinationsStore.GetByName("not-a-real-name")
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(HaveLen(0))

				By("getting by guid")
				destinations, err = egressDestinationsStore.GetByGUID(createdDestinations[0].GUID)
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(HaveLen(1))

				By("getting by nonexistant guid")
				destinations, err = egressDestinationsStore.GetByGUID("not-a-real-guid")
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(HaveLen(0))

				By("updating")
				destinationToUpdate1 := createdDestinations[0]
				destinationToUpdate1.Name = "dest-1-updated"
				destinationToUpdate1.Description = "desc-1-updated"
				destinationToUpdate1.Protocol = "tcp-updated"
				destinationToUpdate1.IPRanges = []store.IPRange{{Start: "2.3.3.3", End: "2.3.3.4"}}
				destinationToUpdate1.Ports = []store.Ports{{Start: 9090, End: 9091}}

				destinationToUpdate2 := createdDestinations[1]
				destinationToUpdate2.Name = "dest-2-updated"
				destinationToUpdate2.Description = "desc-2-updated"
				destinationToUpdate2.Protocol = "icmp-updated"
				destinationToUpdate2.IPRanges = []store.IPRange{{Start: "2.3.3.4", End: "2.3.3.5"}}
				destinationToUpdate2.ICMPType = 15
				destinationToUpdate2.ICMPCode = 16

				updatedDestinations, err := egressDestinationsStore.Update([]store.EgressDestination{destinationToUpdate1, destinationToUpdate2})
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedDestinations).To(HaveLen(2))
				Expect(updatedDestinations).To(Equal([]store.EgressDestination{destinationToUpdate1, destinationToUpdate2}))

				By("updating with an error")
				destinationToUpdate2.GUID = "missing"
				updatedDestinationsWithNoGUID, errWithNoGUID := egressDestinationsStore.Update([]store.EgressDestination{destinationToUpdate1, destinationToUpdate2})
				Expect(errWithNoGUID).To(MatchError("egress destination store update iprange: destination GUID not found"))
				Expect(updatedDestinationsWithNoGUID).To(HaveLen(0))

				By("listing updated destinations to ensure the updates were persisted")
				destinations, err = egressDestinationsStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(ConsistOf(updatedDestinations))

				By("updating a destination that has no metadata row")
				tx, err := realDb.Beginx()
				Expect(err).NotTo(HaveOccurred())

				err = destinationMetadataRepo.Delete(tx, updatedDestinations[1].GUID)
				Expect(err).NotTo(HaveOccurred())

				err = tx.Commit()
				Expect(err).NotTo(HaveOccurred())

				destinationToUpdate2.GUID = updatedDestinations[1].GUID
				_, err = egressDestinationsStore.Update([]store.EgressDestination{destinationToUpdate2})
				Expect(err).NotTo(HaveOccurred())

				By("verifying the destination that had no metadata persisted the update")
				destinations, err = egressDestinationsStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(ConsistOf(updatedDestinations))

				By("deleting")
				deletedDestination, err := egressDestinationsStore.Delete(createdDestinations[0].GUID)
				Expect(err).NotTo(HaveOccurred())
				Expect(deletedDestination).To(Equal(updatedDestinations[0]))

				By("deleting another")
				deletedDestination, err = egressDestinationsStore.Delete(createdDestinations[1].GUID)
				Expect(err).NotTo(HaveOccurred())
				Expect(deletedDestination).To(Equal(updatedDestinations[1]))

				By("asserting all are gone")
				destinations, err = egressDestinationsStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(HaveLen(0))
			})

			Context("when destination metadata returns duplicate name error", func() {
				BeforeEach(func() {
					toBeCreatedDestinations = []store.EgressDestination{
						{
							Name:        "dupe",
							Description: "dupe",
							Protocol:    "tcp",
							IPRanges:    []store.IPRange{{Start: "1.2.2.2", End: "1.2.2.3"}},
							Ports:       []store.Ports{{Start: 8080, End: 8081}},
						},
						{
							Name:        "dupe2",
							Description: "dupe2",
							Protocol:    "tcp",
							IPRanges:    []store.IPRange{{Start: "1.2.2.2", End: "1.2.2.3"}},
							Ports:       []store.Ports{{Start: 8080, End: 8081}},
						},
					}

					var err error
					createdDestinations, err = egressDestinationsStore.Create(toBeCreatedDestinations)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns a specific error when DB detects a duplicate on create", func() {
					_, err := egressDestinationsStore.Create(toBeCreatedDestinations[:1])
					Expect(err).To(MatchError("egress destination store create destination metadata: duplicate name error: entry with name 'dupe' already exists"))
				})

				It("returns a specific error when DB detects a duplicate name on update", func() {
					createdDestinations[1].Name = "dupe"
					_, err := egressDestinationsStore.Update(createdDestinations[1:])
					Expect(err).To(MatchError("egress destination store update destination metadata: duplicate name error: entry with name 'dupe' already exists"))
				})
			})

			Context("when attempting to delete a destination that is referenced by a policy", func() {
				BeforeEach(func() {
					toBeCreatedDestinations := []store.EgressDestination{
						{
							Name:        "dest-1",
							Description: "desc-1",
							Protocol:    "tcp",
							IPRanges:    []store.IPRange{{Start: "1.2.2.2", End: "1.2.2.3"}},
							Ports:       []store.Ports{{Start: 8080, End: 8081}},
						},
					}

					var err error
					createdDestinations, err = egressDestinationsStore.Create(toBeCreatedDestinations)
					Expect(err).NotTo(HaveOccurred())

					toBeCreatedEgressPolicy := []store.EgressPolicy{
						{
							Source: store.EgressSource{
								ID: "some-app-guid",
							},
							Destination: store.EgressDestination{
								GUID: createdDestinations[0].GUID,
							},
						},
					}

					_, err = egressPolicyStore.Create(toBeCreatedEgressPolicy)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns a foreign key error", func() {
					_, err := egressDestinationsStore.Delete(createdDestinations[0].GUID)
					_, ok := err.(store.ForeignKeyError)
					Expect(ok).To(BeTrue(), "expected store.ForeignKeyError, got %v", err)
				})
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

		Context("Update", func() {
			var (
				destinationsToUpdate = []store.EgressDestination{
					{
						Name:        "dupe",
						Description: " ",
						Protocol:    "icmp",
						IPRanges:    []store.IPRange{{Start: "2.2.2.4", End: "2.2.2.5"}},
						ICMPType:    11,
						ICMPCode:    14,
					},
				}
			)
			Context("when the transaction cannot be created", func() {
				BeforeEach(func() {
					mockDB.BeginxReturns(nil, errors.New("can't create a transaction"))
				})

				It("returns an error", func() {
					_, err := egressDestinationsStore.Update(destinationsToUpdate)
					Expect(err).To(MatchError("egress destination store update transaction: can't create a transaction"))
				})
			})

			Context("when getting by guid fails", func() {
				BeforeEach(func() {
					egressDestinationRepo.GetByGUIDReturns(nil, errors.New("something bad happened"))
				})

				It("returns the error", func() {
					_, err := egressDestinationsStore.Update(destinationsToUpdate)
					Expect(err).To(MatchError("egress destination store update GetByGUID: something bad happened"))
				})

				It("rolls back the transaction", func() {
					egressDestinationsStore.Update(destinationsToUpdate)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("when getting by guid returns an unexpected count of destinations", func() {
				BeforeEach(func() {
					egressDestinationRepo.GetByGUIDReturns([]store.EgressDestination{}, nil)
				})

				It("returns the error", func() {
					_, err := egressDestinationsStore.Update(destinationsToUpdate)
					Expect(err).To(MatchError("egress destination store update iprange: destination GUID not found"))
				})

				It("rolls back the transaction", func() {
					egressDestinationsStore.Update(destinationsToUpdate)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})

			Context("when updating the destination metadata fails", func() {
				BeforeEach(func() {
					destinationMetadataRepo.UpsertReturns(errors.New("can't update metadata"))
					egressDestinationRepo.GetByGUIDReturns([]store.EgressDestination{{}}, nil)
				})

				It("rolls back the transaction", func() {
					egressDestinationsStore.Update(destinationsToUpdate)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})

				It("returns the error", func() {
					_, err := egressDestinationsStore.Update(destinationsToUpdate)
					Expect(err).To(MatchError("egress destination store upsert metadata: can't update metadata"))
				})
			})

			Context("when updating the destination fails", func() {
				BeforeEach(func() {
					egressDestinationRepo.UpdateIPRangeReturns(errors.New("can't update iprange"))
					egressDestinationRepo.GetByGUIDReturns([]store.EgressDestination{{}}, nil)
				})

				It("rolls back the transaction", func() {
					egressDestinationsStore.Update(destinationsToUpdate)
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})

				It("returns the error", func() {
					_, err := egressDestinationsStore.Update(destinationsToUpdate)
					Expect(err).To(MatchError("egress destination store update iprange: can't update iprange"))
				})
			})

			Context("when the transaction cannot be committed", func() {
				var err error
				BeforeEach(func() {
					egressDestinationRepo.GetByGUIDReturns([]store.EgressDestination{{}}, nil)
					tx.CommitReturns(errors.New("can't commit transaction"))
					_, err = egressDestinationsStore.Update(destinationsToUpdate)
				})

				It("returns an error", func() {
					Expect(err).To(MatchError("egress destination store update commit transaction: can't commit transaction"))
				})

				It("rolls back the transaction", func() {
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})
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
				var (
					err                  error
					destinationsToCreate []store.EgressDestination
				)

				BeforeEach(func() {
					destinationsToCreate = []store.EgressDestination{
						{
							Name:        "dupe",
							Description: " ",
							Protocol:    "icmp",
							IPRanges:    []store.IPRange{{Start: "2.2.2.4", End: "2.2.2.5"}},
							ICMPType:    11,
							ICMPCode:    14,
						},
					}
				})

				Context("normal error", func() {
					BeforeEach(func() {
						destinationMetadataRepo.UpsertReturns(errors.New("can't create a destination metadata"))
						_, err = egressDestinationsStore.Create(destinationsToCreate)
					})

					It("returns an error", func() {
						Expect(err).To(MatchError("egress destination store create destination metadata: can't create a destination metadata"))
					})

					It("rolls back the transaction", func() {
						Expect(tx.RollbackCallCount()).To(Equal(1))
					})
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
					Expect(err).To(MatchError("egress destination store get all transaction: can't create a transaction"))
				})
			})
		})

		Context("GetByGUID", func() {
			Context("when the transaction cannot be created", func() {
				BeforeEach(func() {
					mockDB.BeginxReturns(nil, errors.New("can't create a transaction"))
				})

				It("returns an error", func() {
					_, err := egressDestinationsStore.GetByGUID("some-guid")
					Expect(err).To(MatchError("egress destination store get by guid transaction: can't create a transaction"))
				})
			})

			Context("when getting the destination from the table fails", func() {
				BeforeEach(func() {
					egressDestinationRepo.GetByGUIDReturns(nil, errors.New("failed to get"))
				})

				It("rolls back the transaction", func() {
					egressDestinationsStore.GetByGUID("some-guid")
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})
			})
		})

		Context("GetByName", func() {
			Context("when the transaction cannot be created", func() {
				BeforeEach(func() {
					mockDB.BeginxReturns(nil, errors.New("can't create a transaction"))
				})

				It("returns an error", func() {
					_, err := egressDestinationsStore.GetByName("some-name")
					Expect(err).To(MatchError("egress destination store get by name transaction: can't create a transaction"))
				})
			})
		})

		Context("Delete", func() {
			var err error
			Context("when the transaction cannot be created", func() {
				BeforeEach(func() {
					mockDB.BeginxReturns(nil, errors.New("can't create a transaction"))
				})

				It("returns an error", func() {
					_, err := egressDestinationsStore.Delete("a-guid")
					Expect(err).To(MatchError("egress destination store delete transaction: can't create a transaction"))
				})
			})

			Context("when getting the destination fails", func() {
				BeforeEach(func() {
					egressDestinationRepo.GetByGUIDReturns([]store.EgressDestination{}, errors.New("can't get the destination"))
					_, err = egressDestinationsStore.Delete("a-guid")
				})

				It("rolls back the transaction", func() {
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})

				It("returns an error", func() {
					Expect(err).To(MatchError("egress destination store get destination by guid: can't get the destination"))
				})
			})

			Context("when deleting the destination fails", func() {
				BeforeEach(func() {
					egressDestinationRepo.DeleteReturns(errors.New("can't delete"))
					_, err = egressDestinationsStore.Delete("a-guid")
				})

				It("rolls back the transaction", func() {
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})

				It("returns an error", func() {
					Expect(err).To(MatchError("egress destination store delete destination: can't delete"))
				})
			})

			Context("when deleting the destination metadata fails", func() {
				BeforeEach(func() {
					destinationMetadataRepo.DeleteReturns(errors.New("can't delete metadata"))
					_, err = egressDestinationsStore.Delete("a-guid")
				})

				It("rolls back the transaction", func() {
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})

				It("returns an error", func() {
					Expect(err).To(MatchError("egress destination store delete destination metadata: can't delete metadata"))
				})
			})

			Context("when deleting the destination terminal fails", func() {
				BeforeEach(func() {
					terminalsRepo.DeleteReturns(errors.New("can't delete terminal"))
					_, err = egressDestinationsStore.Delete("a-guid")
				})

				It("rolls back the transaction", func() {
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})

				It("returns an error", func() {
					Expect(err).To(MatchError("egress destination store delete destination terminal: can't delete terminal"))
				})
			})

			Context("when committing the transaction fails", func() {
				var err error
				BeforeEach(func() {
					tx.CommitReturns(errors.New("can't commit transaction"))
					_, err = egressDestinationsStore.Delete("a-guid")
				})
				It("rolls back the transaction", func() {
					Expect(tx.RollbackCallCount()).To(Equal(1))
				})

				It("returns an error", func() {
					Expect(err).To(MatchError("egress destination store delete destination commit: can't commit transaction"))
				})
			})
		})
	})
})
