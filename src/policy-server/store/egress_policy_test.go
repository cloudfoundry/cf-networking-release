package store_test

import (
	"errors"
	"fmt"
	"policy-server/store"
	"policy-server/store/fakes"
	testhelpers "test-helpers"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	dbfakes "code.cloudfoundry.org/cf-networking-helpers/db/fakes"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Egress Policy Table", func() {
	var (
		dbConf            db.Config
		realDb            *db.ConnWrapper
		mockDb            *fakes.Db
		egressPolicyTable *store.EgressPolicyTable
		terminalsTable    *store.TerminalsTable
		tx                db.Transaction
		fakeGUIDGenerator *fakes.GUIDGenerator
	)

	getMigratedRealDb := func(dbConfig db.Config) (*db.ConnWrapper, db.Transaction) {
		var err error
		testhelpers.CreateDatabase(dbConf)

		logger := lager.NewLogger("Egress Store Test")

		realDb, err = db.NewConnectionPool(dbConf, 200, 0, 60*time.Minute, "Egress Store Test", "Egress Store Test", logger)
		Expect(err).NotTo(HaveOccurred())

		migrate(realDb)
		tx, err = realDb.Beginx()
		Expect(err).NotTo(HaveOccurred())
		return realDb, tx
	}

	setupEgressPolicyStore := func(db store.Database) store.EgressPolicyStore {
		var currentGUID = 0
		fakeGUIDGenerator = &fakes.GUIDGenerator{}
		fakeGUIDGenerator.NewStub = func() string {
			currentGUID++
			return fmt.Sprintf("guid-%d", currentGUID)
		}

		terminalsTable = &store.TerminalsTable{
			Guids: &store.GuidGenerator{},
		}

		egressPolicyTable = &store.EgressPolicyTable{
			Conn:  db,
			Guids: fakeGUIDGenerator,
		}
		return store.EgressPolicyStore{
			EgressPolicyRepo: egressPolicyTable,
			TerminalsRepo:    terminalsTable,
			Conn:             db,
		}
	}

	BeforeEach(func() {
		mockDb = &fakes.Db{}

		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("store_test_node_%d", time.Now().UnixNano())
		dbConf.Timeout = 30
	})

	AfterEach(func() {
		if tx != nil {
			tx.Rollback()
		}
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		testhelpers.RemoveDatabase(dbConf)
	})

	Context("CreateApp", func() {
		It("should create an app and return the ID", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			appTerminalGUID, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())

			id, err := egressPolicyTable.CreateApp(tx, appTerminalGUID, "some-app-guid")
			Expect(err).ToNot(HaveOccurred())

			Expect(id).To(Equal(int64(1)))

			var foundAppGuid string
			row := tx.QueryRow(`SELECT app_guid FROM apps WHERE id = 1`)
			err = row.Scan(&foundAppGuid)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundAppGuid).To(Equal("some-app-guid"))
		})

		It("should return an error if the driver is not supported", func() {
			setupEgressPolicyStore(mockDb)
			fakeTx := &dbfakes.Transaction{}

			fakeTx.DriverNameReturns("db2")

			_, err := egressPolicyTable.CreateApp(fakeTx, "some-term-guid", "some-app-guid")
			Expect(err).To(MatchError("unknown driver: db2"))
		})
	})

	Context("CreateSpace", func() {
		It("should create a space and return the ID", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			spaceTerminalGUID, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())

			id, err := egressPolicyTable.CreateSpace(tx, spaceTerminalGUID, "some-space-guid")
			Expect(err).ToNot(HaveOccurred())

			Expect(id).To(Equal(int64(1)))

			var foundSpaceGuid string
			row := tx.QueryRow(`SELECT space_guid FROM spaces WHERE id = 1`)
			err = row.Scan(&foundSpaceGuid)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundSpaceGuid).To(Equal("some-space-guid"))
		})

		It("should return an error if the driver is not supported", func() {
			setupEgressPolicyStore(mockDb)
			fakeTx := &dbfakes.Transaction{}

			fakeTx.DriverNameReturns("db2")
			_, err := egressPolicyTable.CreateSpace(fakeTx, "some-term-guid", "some-space-guid")
			Expect(err).To(MatchError("unknown driver: db2"))
		})
	})

	Context("CreateIPRange", func() {

		It("should create an iprange and return the ID", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			checkIpRangeFields := func(entryId int, expectedStartIP, expectedEndIP, expectedProtocol string, expectedStartPort, expectedEndPort, expectedIcmpType, expectedIcmpCode int64) {
				var startIP, endIP, protocol string
				var startPort, endPort, icmpType, icmpCode int64

				row := tx.QueryRow(fmt.Sprintf(`SELECT start_ip, end_ip, protocol, start_port, end_port, icmp_type, icmp_code FROM ip_ranges WHERE id = %d`, entryId))
				err := row.Scan(&startIP, &endIP, &protocol, &startPort, &endPort, &icmpType, &icmpCode)

				Expect(err).ToNot(HaveOccurred())
				Expect(startPort).To(Equal(expectedStartPort))
				Expect(endPort).To(Equal(expectedEndPort))
				Expect(startIP).To(Equal(expectedStartIP))
				Expect(endIP).To(Equal(expectedEndIP))
				Expect(protocol).To(Equal(expectedProtocol))
				Expect(icmpType).To(Equal(expectedIcmpType))
				Expect(icmpCode).To(Equal(expectedIcmpCode))
			}

			By("creating an ip range with TCP as protocol")
			ipRangeTerminalGUID, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())
			id, err := egressPolicyTable.CreateIPRange(tx, ipRangeTerminalGUID, "1.1.1.1", "2.2.2.2", "tcp", 8080, 8081, 0, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(int64(1)))

			checkIpRangeFields(1, "1.1.1.1", "2.2.2.2", "tcp", int64(8080), int64(8081), int64(0), int64(0))

			By("creating an ip range with ICMP as protocol")
			ipRangeTerminalGUID, err = terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())
			id, err = egressPolicyTable.CreateIPRange(tx, ipRangeTerminalGUID, "1.1.1.1", "2.2.2.2", "icmp", 0, 0, 2, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(int64(2)))

			checkIpRangeFields(2, "1.1.1.1", "2.2.2.2", "icmp", int64(0), int64(0), int64(2), int64(1))
		})

		It("should return an error if the driver is not supported", func() {
			setupEgressPolicyStore(mockDb)
			fakeTx := &dbfakes.Transaction{}

			fakeTx.DriverNameReturns("db2")

			_, err := egressPolicyTable.CreateIPRange(fakeTx, "some-term-guid", "1.1.1.1", "2.2.2.2", "tcp", 8080, 8081, 0, 0)
			Expect(err).To(MatchError("unknown driver: db2"))
		})
	})

	Context("CreateEgressPolicy", func() {
		It("should create and return the id for an egress policy", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			sourceTerminalId, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())
			destinationTerminalId, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())

			appLifecycle := "running"
			guid, err := egressPolicyTable.CreateEgressPolicy(tx, sourceTerminalId, destinationTerminalId, appLifecycle)
			Expect(err).ToNot(HaveOccurred())
			Expect(guid).To(Equal("guid-1"))

			var foundSourceID, foundDestinationID, foundAppLifecycle string
			row := tx.QueryRow(tx.Rebind(`SELECT source_guid, destination_guid, app_lifecycle FROM egress_policies WHERE guid = ?`), guid)
			err = row.Scan(&foundSourceID, &foundDestinationID, &foundAppLifecycle)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundSourceID).To(Equal(sourceTerminalId))
			Expect(foundDestinationID).To(Equal(destinationTerminalId))
			Expect(foundAppLifecycle).To(Equal(appLifecycle))

			By("checking that if bad args are sent, it returns an error") // merged because db's are slow
			_, err = egressPolicyTable.CreateEgressPolicy(tx, "some-term-guid", "some-term-guid", "banana")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("DeleteEgressPolicy", func() {
		It("deletes the policy", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			sourceTerminalId, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())
			destinationTerminalId, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())

			egressPolicyGUID, err := egressPolicyTable.CreateEgressPolicy(tx, sourceTerminalId, destinationTerminalId, "all")
			Expect(err).ToNot(HaveOccurred())

			err = egressPolicyTable.DeleteEgressPolicy(tx, egressPolicyGUID)
			Expect(err).ToNot(HaveOccurred())

			var policyCount int
			row := tx.QueryRow(tx.Rebind(`SELECT COUNT(guid) FROM egress_policies WHERE guid = ?`), egressPolicyGUID)
			err = row.Scan(&policyCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(policyCount).To(Equal(0))
		})

		It("should return the sql error", func() {
			fakeTx := &dbfakes.Transaction{}
			fakeTx.ExecReturns(nil, errors.New("broke"))

			setupEgressPolicyStore(mockDb)

			err := egressPolicyTable.DeleteEgressPolicy(fakeTx, "some-guid")
			Expect(err).To(MatchError("broke"))
		})
	})

	Context("DeleteIPRange", func() {
		It("deletes the ip range", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			ipRangeTerminalGUID, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())

			ipRangeID, err := egressPolicyTable.CreateIPRange(tx, ipRangeTerminalGUID, "1.1.1.1", "2.2.2.2", "tcp", 8080, 8081, 0, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipRangeID).To(Equal(int64(1)))

			err = egressPolicyTable.DeleteIPRange(tx, ipRangeID)
			Expect(err).ToNot(HaveOccurred())

			var ipRangeCount int
			row := tx.QueryRow(`SELECT COUNT(id) FROM ip_ranges WHERE id = 1`)
			err = row.Scan(&ipRangeCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipRangeCount).To(Equal(0))
		})

		It("should return the sql error", func() {
			setupEgressPolicyStore(mockDb)

			fakeTx := &dbfakes.Transaction{}
			fakeTx.ExecReturns(nil, errors.New("broke"))

			err := egressPolicyTable.DeleteIPRange(fakeTx, 2)
			Expect(err).To(MatchError("broke"))
		})
	})

	Context("DeleteTerminal", func() {
		It("deletes the terminal", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			var err error
			terminalGUID, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())

			err = terminalsTable.Delete(tx, terminalGUID)
			Expect(err).ToNot(HaveOccurred())

			var terminalCount int
			row := tx.QueryRow(tx.Rebind(`SELECT COUNT(guid) FROM terminals WHERE guid = ?`), terminalGUID)
			err = row.Scan(&terminalCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(terminalCount).To(Equal(0))
		})

		It("should return the sql error", func() {
			setupEgressPolicyStore(mockDb)

			fakeTx := &dbfakes.Transaction{}
			fakeTx.ExecReturns(nil, errors.New("broke"))

			err := terminalsTable.Delete(fakeTx, "some-term-guid")
			Expect(err).To(MatchError("broke"))
		})
	})

	Context("DeleteApp", func() {
		It("deletes the app provided a terminal guid", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			appTerminalGUID, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())

			appID, err := egressPolicyTable.CreateApp(tx, appTerminalGUID, "some-app-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(appID).To(Equal(int64(1)))

			err = egressPolicyTable.DeleteApp(tx, appTerminalGUID)
			Expect(err).ToNot(HaveOccurred())

			var appCount int
			row := tx.QueryRow(`SELECT COUNT(id) FROM apps WHERE id = 1`)
			err = row.Scan(&appCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(appCount).To(Equal(0))
		})

		It("should return the sql error", func() {
			setupEgressPolicyStore(mockDb)

			fakeTx := &dbfakes.Transaction{}
			fakeTx.ExecReturns(nil, errors.New("broke"))

			err := egressPolicyTable.DeleteApp(fakeTx, "2")
			Expect(err).To(MatchError("broke"))
		})
	})

	Context("DeleteSpace", func() {
		It("deletes the space provided a terminal guid", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			spaceTerminalGUID, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())

			spaceID, err := egressPolicyTable.CreateSpace(tx, spaceTerminalGUID, "some-space-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(spaceID).To(Equal(int64(1)))

			err = egressPolicyTable.DeleteSpace(tx, spaceTerminalGUID)
			Expect(err).ToNot(HaveOccurred())

			var spaceCount int
			row := tx.QueryRow(`SELECT COUNT(id) FROM spaces WHERE id = 1`)
			err = row.Scan(&spaceCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(spaceCount).To(Equal(0))
		})

		It("should return the sql error", func() {
			setupEgressPolicyStore(mockDb)

			fakeTx := &dbfakes.Transaction{}
			fakeTx.ExecReturns(nil, errors.New("broke"))

			err := egressPolicyTable.DeleteSpace(fakeTx, "a-guid")
			Expect(err).To(MatchError("broke"))
		})
	})

	Context("IsTerminalInUse", func() {
		It("returns true if the terminal is in use by an egress policy", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			destinationTerminalGUID, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())
			sourceTerminalGUID, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())

			_, err = egressPolicyTable.CreateEgressPolicy(tx, sourceTerminalGUID, destinationTerminalGUID, "all")
			Expect(err).ToNot(HaveOccurred())
			inUse, err := egressPolicyTable.IsTerminalInUse(tx, sourceTerminalGUID)
			Expect(err).ToNot(HaveOccurred())
			Expect(inUse).To(BeTrue())

			By("returns false if the terminal is not in use by an egress policy") //combined because db's are slow
			inUse, err = egressPolicyTable.IsTerminalInUse(tx, "some-term-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(inUse).To(BeFalse())
		})
	})

	Context("when retrieving egress policies", func() {
		var (
			egressPolicies            []store.EgressPolicy
			createdEgressPolicies     []store.EgressPolicy
			egressDestinations        []store.EgressDestination
			createdEgressDestinations []store.EgressDestination
		)
		Context("when the APIs succeed", func() {
			BeforeEach(func() {
				db, _ := getMigratedRealDb(dbConf)
				egressStore := setupEgressPolicyStore(db)

				var err error

				egressDestinations = []store.EgressDestination{
					{
						Name:        "a",
						Description: "desc a",
						Protocol:    "tcp",
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
					{
						Name:        "b",
						Description: "desc b",
						Protocol:    "udp",
						IPRanges: []store.IPRange{
							{
								Start: "2.2.3.4",
								End:   "2.2.3.5",
							},
						},
					},
					{
						Name:        "c",
						Description: "desc c",
						Protocol:    "icmp",
						ICMPType:    1,
						ICMPCode:    2,
						IPRanges: []store.IPRange{
							{
								Start: "2.2.3.4",
								End:   "2.2.3.5",
							},
						},
					},
					{
						Name:        "old-entry",
						Description: "this represents an entry that has no destination_metadata",
						Protocol:    "icmp",
						ICMPType:    1,
						ICMPCode:    2,
						IPRanges: []store.IPRange{
							{
								Start: "2.2.3.4",
								End:   "2.2.3.5",
							},
						},
					},
				}

				destinationStore := egressDestinationStore(db)
				createdEgressDestinations, err = destinationStore.Create(egressDestinations)
				Expect(err).ToNot(HaveOccurred())
				// delete one of the description_metadatas to simulate destinations that were created before the
				// destination_metadatas table existed
				_, err = db.Exec(`DELETE FROM destination_metadatas WHERE name='old-entry';`)
				Expect(err).ToNot(HaveOccurred())

				egressPolicies = []store.EgressPolicy{
					{
						Source: store.EgressSource{
							ID:   "some-app-guid",
							Type: "app",
						},
						Destination: store.EgressDestination{
							GUID: createdEgressDestinations[0].GUID,
						},
					},
					{
						Source: store.EgressSource{
							ID:   "space-guid",
							Type: "space",
						},
						Destination: store.EgressDestination{
							GUID: createdEgressDestinations[1].GUID,
						},
					},
					{
						Source: store.EgressSource{
							ID:   "different-app-guid",
							Type: "app",
						},
						Destination: store.EgressDestination{
							GUID: createdEgressDestinations[2].GUID,
						},
					},
					{
						Source: store.EgressSource{
							ID:   "different-space-guid",
							Type: "space",
						},
						Destination: store.EgressDestination{
							GUID: createdEgressDestinations[3].GUID,
						},
					},
				}

				createdEgressPolicies, err = egressStore.Create(egressPolicies)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("GetByGUID", func() {
				It("should return the requeted egress policies", func() {
					egressPolicies, err := egressPolicyTable.GetByGUID(tx, createdEgressPolicies[0].ID, createdEgressPolicies[1].ID)
					Expect(err).NotTo(HaveOccurred())
					Expect(egressPolicies).To(ConsistOf(
						store.EgressPolicy{
							ID: createdEgressPolicies[0].ID,
							Source: store.EgressSource{
								Type:         "app",
								TerminalGUID: createdEgressPolicies[0].Source.TerminalGUID,
								ID:           "some-app-guid",
							},
							Destination: createdEgressDestinations[0],
						},
						store.EgressPolicy{
							ID: createdEgressPolicies[1].ID,
							Source: store.EgressSource{
								Type:         "space",
								TerminalGUID: createdEgressPolicies[1].Source.TerminalGUID,
								ID:           "space-guid",
							},
							Destination: createdEgressDestinations[1],
						}))
				})

				Context("when a non-existent policy/no policy guid is requested", func() {
					It("returns an empty array", func() {
						egressPolicies, err := egressPolicyTable.GetByGUID(tx, "what-policy?")
						Expect(err).ToNot(HaveOccurred())
						Expect(egressPolicies).To(HaveLen(0))

						egressPolicies, err = egressPolicyTable.GetByGUID(tx)
						Expect(err).ToNot(HaveOccurred())
						Expect(egressPolicies).To(HaveLen(0))
					})
				})
			})

			Context("GetAllPolicies", func() {
				It("returns policies", func() {
					listedPolicies, err := egressPolicyTable.GetAllPolicies()
					Expect(err).ToNot(HaveOccurred())
					Expect(listedPolicies).To(HaveLen(4))
					Expect(listedPolicies).To(ConsistOf([]store.EgressPolicy{
						{
							ID: "guid-1",
							Source: store.EgressSource{
								ID:           "some-app-guid",
								Type:         "app",
								TerminalGUID: createdEgressPolicies[0].Source.TerminalGUID,
							},
							Destination: store.EgressDestination{
								GUID:        createdEgressDestinations[0].GUID,
								Name:        "a",
								Description: "desc a",
								Protocol:    "tcp",
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
							ID: "guid-2",
							Source: store.EgressSource{
								ID:           "space-guid",
								Type:         "space",
								TerminalGUID: createdEgressPolicies[1].Source.TerminalGUID,
							},
							Destination: store.EgressDestination{
								GUID:        createdEgressDestinations[1].GUID,
								Name:        "b",
								Description: "desc b",
								Protocol:    "udp",
								IPRanges: []store.IPRange{
									{
										Start: "2.2.3.4",
										End:   "2.2.3.5",
									},
								},
							},
						},
						{
							ID: "guid-3",
							Source: store.EgressSource{
								ID:           "different-app-guid",
								Type:         "app",
								TerminalGUID: createdEgressPolicies[2].Source.TerminalGUID,
							},
							Destination: store.EgressDestination{
								GUID:        createdEgressDestinations[2].GUID,
								Name:        "c",
								Description: "desc c",
								Protocol:    "icmp",
								ICMPType:    1,
								ICMPCode:    2,
								IPRanges: []store.IPRange{
									{
										Start: "2.2.3.4",
										End:   "2.2.3.5",
									},
								},
							},
						},
						{
							ID: "guid-4",
							Source: store.EgressSource{
								ID:           "different-space-guid",
								Type:         "space",
								TerminalGUID: createdEgressPolicies[3].Source.TerminalGUID,
							},
							Destination: store.EgressDestination{
								GUID:        createdEgressDestinations[3].GUID,
								Name:        "",
								Description: "",
								Protocol:    "icmp",
								ICMPType:    1,
								ICMPCode:    2,
								IPRanges: []store.IPRange{
									{
										Start: "2.2.3.4",
										End:   "2.2.3.5",
									},
								},
							},
						},
					}))
				})
			})
		})
		Context("when the query fails", func() {
			It("returns an error", func() {
				setupEgressPolicyStore(mockDb)

				mockDb.QueryReturns(nil, errors.New("some error that sql would return"))

				egressPolicyTable = &store.EgressPolicyTable{
					Conn: mockDb,
				}

				_, err := egressPolicyTable.GetAllPolicies()
				Expect(err).To(MatchError("some error that sql would return"))
			})
		})
	})

	Context("GetTerminalByAppGUID", func() {
		It("should return the terminal id for an app if it exists", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			terminalId, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())
			_, err = egressPolicyTable.CreateApp(tx, terminalId, "some-app-guid")
			Expect(err).ToNot(HaveOccurred())

			foundID, err := egressPolicyTable.GetTerminalByAppGUID(tx, "some-app-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(foundID).To(Equal(terminalId))

			By("should return empty string and no error if the app is not found")
			foundID, err = egressPolicyTable.GetTerminalByAppGUID(tx, "garbage-app-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(foundID).To(Equal(""))
		})
	})

	Context("GetTerminalBySpaceGUID", func() {
		It("should return the terminal guid for a space if it exists", func() {
			db, tx := getMigratedRealDb(dbConf)
			setupEgressPolicyStore(db)

			terminalId, err := terminalsTable.Create(tx)
			Expect(err).ToNot(HaveOccurred())
			_, err = egressPolicyTable.CreateSpace(tx, terminalId, "some-space-guid")
			Expect(err).ToNot(HaveOccurred())

			foundID, err := egressPolicyTable.GetTerminalBySpaceGUID(tx, "some-space-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(foundID).To(Equal(terminalId))

			By("should return empty string and no error if the space is not found")
			foundID, err = egressPolicyTable.GetTerminalBySpaceGUID(tx, "garbage-space-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(foundID).To(Equal(""))
		})
	})

	Context("GetBySourceGuids", func() {
		Context("When using a real db", func() {
			var (
				egressPolicies        []store.EgressPolicy
				createdDestinations   []store.EgressDestination
				createdEgressPolicies []store.EgressPolicy
			)

			BeforeEach(func() {
				db, _ := getMigratedRealDb(dbConf)
				egressStore := setupEgressPolicyStore(db)

				egressDestinations := []store.EgressDestination{
					{
						Name:     "a",
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
					{
						Name:     "b",
						Protocol: "udp",
						IPRanges: []store.IPRange{
							{
								Start: "2.2.3.4",
								End:   "2.2.3.5",
							},
						},
					},
					{
						Name:     "c",
						Protocol: "icmp",
						ICMPType: 1,
						ICMPCode: 2,
						IPRanges: []store.IPRange{
							{
								Start: "2.2.3.4",
								End:   "2.2.3.5",
							},
						},
					},
					{
						Name:     "d",
						Protocol: "udp",
						Ports: []store.Ports{
							{
								Start: 8080,
								End:   8081,
							},
						},
						IPRanges: []store.IPRange{
							{
								Start: "3.2.3.4",
								End:   "3.2.3.5",
							},
						},
					},
					{
						Name:     "e",
						Protocol: "udp",
						IPRanges: []store.IPRange{
							{
								Start: "2.2.3.4",
								End:   "2.2.3.5",
							},
						},
					},
				}

				var err error
				createdDestinations, err = egressDestinationStore(db).Create(egressDestinations)
				Expect(err).ToNot(HaveOccurred())

				egressPolicies = []store.EgressPolicy{
					{
						Source: store.EgressSource{
							ID:   "some-app-guid",
							Type: "app",
						},
						Destination: store.EgressDestination{
							GUID: createdDestinations[0].GUID,
						},
					},
					{
						Source: store.EgressSource{
							ID:   "different-app-guid",
							Type: "app",
						},
						Destination: store.EgressDestination{
							GUID: createdDestinations[1].GUID,
						},
					},
					{
						Source: store.EgressSource{
							ID:   "different-app-guid",
							Type: "app",
						},
						Destination: store.EgressDestination{
							GUID: createdDestinations[2].GUID,
						},
					},
					{
						Source: store.EgressSource{
							ID:   "some-space-guid",
							Type: "space",
						},
						Destination: store.EgressDestination{
							GUID: createdDestinations[3].GUID,
						},
					},
					{
						Source: store.EgressSource{
							ID: "never-referenced-app-guid",
						},
						Destination: store.EgressDestination{
							GUID: createdDestinations[4].GUID,
						},
					},
				}
				createdEgressPolicies, err = egressStore.Create(egressPolicies)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when there are policies with the given id", func() {
				It("returns egress policies with those ids", func() {
					By("returning egress policies with existing ids")
					policies, err := egressPolicyTable.GetBySourceGuids([]string{"some-app-guid", "different-app-guid", "some-space-guid"})
					Expect(err).ToNot(HaveOccurred())

					// egressStore.Create doesn't return the full destination, but GetBySourceGuids does
					expectedEgressPolicies := createdEgressPolicies[:4]
					expectedEgressPolicies[0].Destination = createdDestinations[0]
					expectedEgressPolicies[1].Destination = createdDestinations[1]
					expectedEgressPolicies[2].Destination = createdDestinations[2]
					expectedEgressPolicies[3].Destination = createdDestinations[3]
					Expect(policies).To(ConsistOf(expectedEgressPolicies))

					By("returning empty list for non-existent ids")
					policies, err = egressPolicyTable.GetBySourceGuids([]string{"meow-this-is-a-bogus-app-guid"})
					Expect(err).ToNot(HaveOccurred())
					Expect(policies).To(HaveLen(0))
				})
			})
		})

		Context("when the query fails", func() {
			It("returns an error", func() {
				setupEgressPolicyStore(mockDb)

				mockDb.QueryReturns(nil, errors.New("some error that sql would return"))

				egressPolicyTable = &store.EgressPolicyTable{
					Conn: mockDb,
				}

				_, err := egressPolicyTable.GetBySourceGuids([]string{"id-does-not-matter"})
				Expect(err).To(MatchError("some error that sql would return"))
			})
		})
	})

	Context("GetByFilter", func() {
		Context("When using a real db", func() {
			var (
				egressPolicies        []store.EgressPolicy
				createdDestinations   []store.EgressDestination
				createdEgressPolicies []store.EgressPolicy
			)

			BeforeEach(func() {
				db, _ := getMigratedRealDb(dbConf)
				egressStore := setupEgressPolicyStore(db)

				egressDestinations := []store.EgressDestination{
					{
						Name:     "a",
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
					{
						Name:     "b",
						Protocol: "udp",
						IPRanges: []store.IPRange{
							{
								Start: "2.2.3.4",
								End:   "2.2.3.5",
							},
						},
					},
					{
						Name:     "c",
						Protocol: "icmp",
						ICMPType: 1,
						ICMPCode: 2,
						IPRanges: []store.IPRange{
							{
								Start: "2.2.3.4",
								End:   "2.2.3.5",
							},
						},
					},
					{
						Name:     "d",
						Protocol: "udp",
						Ports: []store.Ports{
							{
								Start: 8080,
								End:   8081,
							},
						},
						IPRanges: []store.IPRange{
							{
								Start: "3.2.3.4",
								End:   "3.2.3.5",
							},
						},
					},
					{
						Name:     "e",
						Protocol: "udp",
						IPRanges: []store.IPRange{
							{
								Start: "2.2.3.4",
								End:   "2.2.3.5",
							},
						},
					},
				}

				var err error
				createdDestinations, err = egressDestinationStore(db).Create(egressDestinations)
				Expect(err).ToNot(HaveOccurred())

				egressPolicies = []store.EgressPolicy{
					{
						Source: store.EgressSource{
							ID:   "some-app-guid",
							Type: "app",
						},
						Destination: store.EgressDestination{
							GUID: createdDestinations[0].GUID,
						},
						AppLifecycle: "running",
					},
					{
						Source: store.EgressSource{
							ID:   "different-app-guid",
							Type: "app",
						},
						Destination: store.EgressDestination{
							GUID: createdDestinations[1].GUID,
						},
						AppLifecycle: "all",
					},
					{
						Source: store.EgressSource{
							ID:   "different-app-guid",
							Type: "app",
						},
						Destination: store.EgressDestination{
							GUID: createdDestinations[2].GUID,
						},
						AppLifecycle: "running",
					},
					{
						Source: store.EgressSource{
							ID:   "some-space-guid",
							Type: "space",
						},
						Destination: store.EgressDestination{
							GUID: createdDestinations[3].GUID,
						},
						AppLifecycle: "running",
					},
					{
						Source: store.EgressSource{
							ID:   "some-space-guid",
							Type: "space",
						},
						Destination: store.EgressDestination{
							GUID: createdDestinations[3].GUID,
						},
						AppLifecycle: "staging",
					},
					{
						Source: store.EgressSource{
							ID: "never-referenced-app-guid",
						},
						Destination: store.EgressDestination{
							GUID: createdDestinations[4].GUID,
						},
						AppLifecycle: "running",
					},
				}
				createdEgressPolicies, err = egressStore.Create(egressPolicies)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when there are policies with the filter", func() {
				It("returns egress policies that match", func() {
					By("returning egress policies by sourceID")
					policies, err := egressPolicyTable.GetByFilter([]string{"different-app-guid"}, []string{}, []string{}, []string{}, []string{})
					Expect(err).ToNot(HaveOccurred())

					// egressStore.Create doesn't return the full destination, but GetBySourceGuids does
					expectedEgressPolicies := createdEgressPolicies[1:3]
					expectedEgressPolicies[0].Destination = createdDestinations[1]
					expectedEgressPolicies[1].Destination = createdDestinations[2]
					Expect(policies).To(ConsistOf(expectedEgressPolicies))

					By("returning egress policies by sourceType")
					policies, err = egressPolicyTable.GetByFilter([]string{}, []string{"app"}, []string{}, []string{}, []string{})
					Expect(err).ToNot(HaveOccurred())

					// egressStore.Create doesn't return the full destination, but GetBySourceGuids does
					expectedEgressPolicies = createdEgressPolicies[0:3]
					expectedEgressPolicies = append([]store.EgressPolicy(nil), expectedEgressPolicies...)
					expectedEgressPolicies = append(expectedEgressPolicies, createdEgressPolicies[5])
					expectedEgressPolicies[0].Destination = createdDestinations[0]
					expectedEgressPolicies[1].Destination = createdDestinations[1]
					expectedEgressPolicies[2].Destination = createdDestinations[2]
					expectedEgressPolicies[3].Destination = createdDestinations[4]
					expectedEgressPolicies[3].Source.Type = "app"
					Expect(policies).To(ConsistOf(expectedEgressPolicies))

					By("returning egress policies sourceID where source is a space")
					policies, err = egressPolicyTable.GetByFilter([]string{"some-space-guid"}, []string{}, []string{}, []string{}, []string{})
					Expect(err).ToNot(HaveOccurred())

					// egressStore.Create doesn't return the full destination, but GetBySourceGuids does
					expectedEgressPolicies = createdEgressPolicies[3:5]
					expectedEgressPolicies[0].Destination = createdDestinations[3]
					expectedEgressPolicies[1].Destination = createdDestinations[3]
					Expect(policies).To(ConsistOf(expectedEgressPolicies))

					By("returning egress policies sourceType where type is space")
					policies, err = egressPolicyTable.GetByFilter([]string{}, []string{"space"}, []string{}, []string{}, []string{})
					Expect(err).ToNot(HaveOccurred())

					// egressStore.Create doesn't return the full destination, but GetBySourceGuids does
					expectedEgressPolicies = createdEgressPolicies[3:5]
					expectedEgressPolicies[0].Destination = createdDestinations[3]
					expectedEgressPolicies[1].Destination = createdDestinations[3]
					Expect(policies).To(ConsistOf(expectedEgressPolicies))

					By("returning egress policies sourceId and sourceType")
					policies, err = egressPolicyTable.GetByFilter([]string{"some-space-guid"}, []string{"space"}, []string{}, []string{}, []string{})
					Expect(err).ToNot(HaveOccurred())

					// egressStore.Create doesn't return the full destination, but GetBySourceGuids does
					expectedEgressPolicies = createdEgressPolicies[3:5]
					expectedEgressPolicies[0].Destination = createdDestinations[3]
					expectedEgressPolicies[1].Destination = createdDestinations[3]
					Expect(policies).To(ConsistOf(expectedEgressPolicies))

					By("returning egress policies with matching app lifecycle")
					policies, err = egressPolicyTable.GetByFilter([]string{}, []string{}, []string{}, []string{}, []string{"staging"})
					Expect(err).ToNot(HaveOccurred())

					// egressStore.Create doesn't return the full destination, but GetBySourceGuids does
					expectedEgressPolicy := createdEgressPolicies[4]
					expectedEgressPolicy.Destination = createdDestinations[3]
					Expect(policies).To(ConsistOf(expectedEgressPolicy))

					By("returning egress policies by destinationId")
					policies, err = egressPolicyTable.GetByFilter([]string{}, []string{}, []string{createdDestinations[0].GUID}, []string{}, []string{})
					Expect(err).ToNot(HaveOccurred())

					// egressStore.Create doesn't return the full destination, but GetBySourceGuids does
					expectedEgressPolicy = createdEgressPolicies[0]
					expectedEgressPolicy.Destination = createdDestinations[0]
					Expect(policies).To(ConsistOf(expectedEgressPolicy))

					By("returning egress policies by destinationName")
					policies, err = egressPolicyTable.GetByFilter([]string{}, []string{}, []string{}, []string{"a"}, []string{})
					Expect(err).ToNot(HaveOccurred())

					// egressStore.Create doesn't return the full destination, but GetBySourceGuids does
					expectedEgressPolicy = createdEgressPolicies[0]
					expectedEgressPolicy.Destination = createdDestinations[0]
					Expect(policies).To(ConsistOf(expectedEgressPolicy))

					By("returning empty list for non-existent ids")
					policies, err = egressPolicyTable.GetByFilter([]string{"meow-bogus"}, []string{}, []string{}, []string{}, []string{})
					Expect(err).ToNot(HaveOccurred())
					Expect(policies).To(HaveLen(0))
				})
			})
		})

	})

	Context("when the query fails", func() {
		It("returns an error", func() {
			setupEgressPolicyStore(mockDb)

			mockDb.QueryReturns(nil, errors.New("some error that sql would return"))

			egressPolicyTable = &store.EgressPolicyTable{
				Conn: mockDb,
			}

			_, err := egressPolicyTable.GetByFilter([]string{"epic fail"}, []string{""}, []string{""}, []string{""}, []string{""})
			Expect(err).To(MatchError("some error that sql would return"))
		})
	})
})

func egressDestinationStore(db store.Database) *store.EgressDestinationStore {
	terminalsRepo := &store.TerminalsTable{
		Guids: &store.GuidGenerator{},
	}

	destinationMetadataTable := &store.DestinationMetadataTable{}
	egressDestinationStore := &store.EgressDestinationStore{
		Conn: db,
		EgressDestinationRepo:   &store.EgressDestinationTable{},
		TerminalsRepo:           terminalsRepo,
		DestinationMetadataRepo: destinationMetadataTable,
	}

	return egressDestinationStore
}
