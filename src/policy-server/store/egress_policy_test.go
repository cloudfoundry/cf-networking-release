package store_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"
	"fmt"
	"policy-server/db"
	"policy-server/store"
	"policy-server/store/fakes"
	"policy-server/store/migrations"
	"test-helpers"
	"time"

	dbfakes "policy-server/db/fakes"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"
)

var _ = Describe("Egress Policy Table", func() {
	var (
		dbConf            dbHelper.Config
		realDb            *db.ConnWrapper
		mockDb            *fakes.Db
		migrator          *migrations.Migrator
		egressPolicyTable *store.EgressPolicyTable
		tx                db.Transaction
		egressStore       store.EgressPolicyStore
	)

	BeforeEach(func() {
		var err error
		mockDb = &fakes.Db{}

		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("store_test_node_%d", time.Now().UnixNano())
		dbConf.Timeout = 30
		testhelpers.CreateDatabase(dbConf)

		logger := lager.NewLogger("Egress Store Test")

		realDb = db.NewConnectionPool(dbConf, 200, 200, "Egress Store Test", "Egress Store Test", logger)
		migrator = &migrations.Migrator{
			MigrateAdapter: &migrations.MigrateAdapter{},
			MigrationsProvider: &migrations.MigrationsProvider{
				Store: &store.MigrationsStore{
					DBConn: realDb,
				},
			},
		}
		_, err = migrator.PerformMigrations(realDb.DriverName(), realDb, 0)
		Expect(err).NotTo(HaveOccurred())

		egressPolicyTable = &store.EgressPolicyTable{
			Conn: realDb,
		}

		egressStore = store.EgressPolicyStore{
			EgressPolicyRepo: egressPolicyTable,
		}
		tx, err = realDb.Beginx()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		tx.Rollback()
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		testhelpers.RemoveDatabase(dbConf)
	})

	Context("CreateTerminal", func() {
		It("should create a Terminal and return the ID", func() {
			id, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			Expect(id).To(Equal(int64(1)))
		})

		It("should return an error if the driver is not supported", func() {
			fakeTx := &dbfakes.Transaction{}

			fakeTx.DriverNameReturns("db2")

			_, err := egressPolicyTable.CreateTerminal(fakeTx)
			Expect(err).To(MatchError("unknown driver: db2"))
		})
	})

	Context("CreateApp", func() {
		It("should create an app and return the ID", func() {
			appTerminalID, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			id, err := egressPolicyTable.CreateApp(tx, appTerminalID, "some-app-guid")
			Expect(err).ToNot(HaveOccurred())

			Expect(id).To(Equal(int64(1)))

			var foundAppGuid string
			row := tx.QueryRow(`SELECT app_guid FROM apps WHERE id = 1`)
			err = row.Scan(&foundAppGuid)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundAppGuid).To(Equal("some-app-guid"))
		})

		It("should return an error if the driver is not supported", func() {
			fakeTx := &dbfakes.Transaction{}

			fakeTx.DriverNameReturns("db2")

			_, err := egressPolicyTable.CreateApp(fakeTx, 1, "some-app-guid")
			Expect(err).To(MatchError("unknown driver: db2"))
		})
	})

	Context("CreateSpace", func() {
		It("should create a space and return the ID", func() {
			spaceTerminalID, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			id, err := egressPolicyTable.CreateSpace(tx, spaceTerminalID, "some-space-guid")
			Expect(err).ToNot(HaveOccurred())

			Expect(id).To(Equal(int64(1)))

			var foundSpaceGuid string
			row := tx.QueryRow(`SELECT space_guid FROM spaces WHERE id = 1`)
			err = row.Scan(&foundSpaceGuid)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundSpaceGuid).To(Equal("some-space-guid"))
		})

		It("should return an error if the driver is not supported", func() {
			fakeTx := &dbfakes.Transaction{}

			fakeTx.DriverNameReturns("db2")

			_, err := egressPolicyTable.CreateSpace(fakeTx, 1, "some-space-guid")
			Expect(err).To(MatchError("unknown driver: db2"))
		})
	})

	Context("CreateIPRange", func() {
		It("should create an iprange and return the ID", func() {
			ipRangeTerminalID, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			id, err := egressPolicyTable.CreateIPRange(tx, ipRangeTerminalID, "1.1.1.1", "2.2.2.2", "tcp", 8080, 8081, 0, 0)
			Expect(err).ToNot(HaveOccurred())

			Expect(id).To(Equal(int64(1)))

			var startIP, endIP, protocol string
			var startPort, endPort, icmpType, icmpCode int64
			row := tx.QueryRow(`SELECT start_ip, end_ip, protocol, start_port, end_port, icmp_type, icmp_code FROM ip_ranges WHERE id = 1`)
			err = row.Scan(&startIP, &endIP, &protocol, &startPort, &endPort, &icmpType, &icmpCode)
			Expect(err).ToNot(HaveOccurred())
			Expect(startPort).To(Equal(int64(8080)))
			Expect(endPort).To(Equal(int64(8081)))
			Expect(startIP).To(Equal("1.1.1.1"))
			Expect(endIP).To(Equal("2.2.2.2"))
			Expect(protocol).To(Equal("tcp"))
			Expect(icmpType).To(Equal(int64(0)))
			Expect(icmpCode).To(Equal(int64(0)))
		})

		It("should create an iprange with icmp and return the ID", func() {
			ipRangeTerminalID, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			id, err := egressPolicyTable.CreateIPRange(tx, ipRangeTerminalID, "1.1.1.1", "2.2.2.2", "icmp", 0, 0, 2, 1)
			Expect(err).ToNot(HaveOccurred())

			Expect(id).To(Equal(int64(1)))

			var startIP, endIP, protocol string
			var startPort, endPort, icmpType, icmpCode int64
			row := tx.QueryRow(`SELECT start_ip, end_ip, protocol, start_port, end_port, icmp_type, icmp_code FROM ip_ranges WHERE id = 1`)
			err = row.Scan(&startIP, &endIP, &protocol, &startPort, &endPort, &icmpType, &icmpCode)
			Expect(err).ToNot(HaveOccurred())
			Expect(startPort).To(Equal(int64(0)))
			Expect(endPort).To(Equal(int64(0)))
			Expect(startIP).To(Equal("1.1.1.1"))
			Expect(endIP).To(Equal("2.2.2.2"))
			Expect(protocol).To(Equal("icmp"))
			Expect(icmpType).To(Equal(int64(2)))
			Expect(icmpCode).To(Equal(int64(1)))
		})

		It("should return an error if the driver is not supported", func() {
			fakeTx := &dbfakes.Transaction{}

			fakeTx.DriverNameReturns("db2")

			_, err := egressPolicyTable.CreateIPRange(fakeTx, 1, "1.1.1.1", "2.2.2.2", "tcp", 8080, 8081, 0, 0)
			Expect(err).To(MatchError("unknown driver: db2"))
		})
	})

	Context("CreateEgressPolicy", func() {
		It("should create and return the id for an egress policy", func() {
			sourceTerminalId, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())
			destinationTerminalId, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			id, err := egressPolicyTable.CreateEgressPolicy(tx, sourceTerminalId, destinationTerminalId)
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(int64(1)))

			var foundSourceID, foundDestinationID int64
			row := tx.QueryRow(`SELECT source_id, destination_id FROM egress_policies WHERE id = 1`)
			err = row.Scan(&foundSourceID, &foundDestinationID)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundSourceID).To(Equal(sourceTerminalId))
			Expect(foundDestinationID).To(Equal(destinationTerminalId))

		})

		It("should return the sql error", func() {
			_, err := egressPolicyTable.CreateEgressPolicy(tx, 2, 3)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("DeleteEgressPolicy", func() {
		var (
			egressPolicyID int64
		)

		BeforeEach(func() {
			sourceTerminalId, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())
			destinationTerminalId, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			egressPolicyID, err = egressPolicyTable.CreateEgressPolicy(tx, sourceTerminalId, destinationTerminalId)
			Expect(err).ToNot(HaveOccurred())
		})

		It("deletes the policy", func() {
			err := egressPolicyTable.DeleteEgressPolicy(tx, egressPolicyID)
			Expect(err).ToNot(HaveOccurred())

			var policyCount int
			row := tx.QueryRow(`SELECT COUNT(id) FROM egress_policies WHERE id = 1`)
			err = row.Scan(&policyCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(policyCount).To(Equal(0))
		})

		It("should return the sql error", func() {
			fakeTx := &dbfakes.Transaction{}
			fakeTx.ExecReturns(nil, errors.New("broke"))

			err := egressPolicyTable.DeleteEgressPolicy(fakeTx, 2)
			Expect(err).To(MatchError("broke"))
		})
	})

	Context("DeleteIPRange", func() {
		var (
			ipRangeID int64
		)

		BeforeEach(func() {
			ipRangeTerminalID, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			ipRangeID, err = egressPolicyTable.CreateIPRange(tx, ipRangeTerminalID, "1.1.1.1", "2.2.2.2", "tcp", 8080, 8081, 0, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipRangeID).To(Equal(int64(1)))
		})

		It("deletes the ip range", func() {
			err := egressPolicyTable.DeleteIPRange(tx, ipRangeID)
			Expect(err).ToNot(HaveOccurred())

			var ipRangeCount int
			row := tx.QueryRow(`SELECT COUNT(id) FROM ip_ranges WHERE id = 1`)
			err = row.Scan(&ipRangeCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipRangeCount).To(Equal(0))
		})

		It("should return the sql error", func() {
			fakeTx := &dbfakes.Transaction{}
			fakeTx.ExecReturns(nil, errors.New("broke"))

			err := egressPolicyTable.DeleteIPRange(fakeTx, 2)
			Expect(err).To(MatchError("broke"))
		})
	})

	Context("DeleteTerminal", func() {
		var (
			terminalID int64
		)

		BeforeEach(func() {
			var err error
			terminalID, err = egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(terminalID).To(Equal(int64(1)))
		})

		It("deletes the terminal", func() {
			err := egressPolicyTable.DeleteTerminal(tx, terminalID)
			Expect(err).ToNot(HaveOccurred())

			var terminalCount int
			row := tx.QueryRow(`SELECT COUNT(id) FROM terminals WHERE id = 1`)
			err = row.Scan(&terminalCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(terminalCount).To(Equal(0))
		})

		It("should return the sql error", func() {
			fakeTx := &dbfakes.Transaction{}
			fakeTx.ExecReturns(nil, errors.New("broke"))

			err := egressPolicyTable.DeleteTerminal(fakeTx, 2)
			Expect(err).To(MatchError("broke"))
		})
	})

	Context("DeleteApp", func() {
		var (
			appID int64
		)

		BeforeEach(func() {
			appTerminalID, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			appID, err = egressPolicyTable.CreateApp(tx, appTerminalID, "some-app-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(appID).To(Equal(int64(1)))

		})

		It("deletes the app", func() {
			err := egressPolicyTable.DeleteApp(tx, appID)
			Expect(err).ToNot(HaveOccurred())

			var appCount int
			row := tx.QueryRow(`SELECT COUNT(id) FROM apps WHERE id = 1`)
			err = row.Scan(&appCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(appCount).To(Equal(0))
		})

		It("should return the sql error", func() {
			fakeTx := &dbfakes.Transaction{}
			fakeTx.ExecReturns(nil, errors.New("broke"))

			err := egressPolicyTable.DeleteApp(fakeTx, 2)
			Expect(err).To(MatchError("broke"))
		})
	})

	Context("DeleteSpace", func() {
		var (
			spaceID int64
		)

		BeforeEach(func() {
			spaceTerminalID, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			spaceID, err = egressPolicyTable.CreateSpace(tx, spaceTerminalID, "some-space-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(spaceID).To(Equal(int64(1)))

		})

		It("deletes the space", func() {
			err := egressPolicyTable.DeleteSpace(tx, spaceID)
			Expect(err).ToNot(HaveOccurred())

			var spaceCount int
			row := tx.QueryRow(`SELECT COUNT(id) FROM spaces WHERE id = 1`)
			err = row.Scan(&spaceCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(spaceCount).To(Equal(0))
		})

		It("should return the sql error", func() {
			fakeTx := &dbfakes.Transaction{}
			fakeTx.ExecReturns(nil, errors.New("broke"))

			err := egressPolicyTable.DeleteSpace(fakeTx, 2)
			Expect(err).To(MatchError("broke"))
		})
	})

	Context("IsTerminalInUse", func() {
		var (
			sourceTerminalID int64
		)

		BeforeEach(func() {
			destinationTerminalID, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())
			sourceTerminalID, err = egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			_, err = egressPolicyTable.CreateEgressPolicy(tx, sourceTerminalID, destinationTerminalID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns true if the terminal is in use by an egress policy", func() {
			inUse, err := egressPolicyTable.IsTerminalInUse(tx, sourceTerminalID)
			Expect(err).ToNot(HaveOccurred())
			Expect(inUse).To(BeTrue())
		})

		It("returns false if the terminal is not in use by an egress policy", func() {
			inUse, err := egressPolicyTable.IsTerminalInUse(tx, 42)
			Expect(err).ToNot(HaveOccurred())
			Expect(inUse).To(BeFalse())
		})
	})

	Context("GetIDCollectionsByEgressPolicy", func() {
		var (
			egressPolicy          store.EgressPolicy
			sourceTerminalID      int64
			destinationTerminalID int64
			egressPolicyID        int64
			appID                 int64
			ipRangeID             int64
		)

		BeforeEach(func() {
			var err error
			egressPolicy = store.EgressPolicy{
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
							Start: "1.1.1.1",
							End:   "2.2.2.2",
						},
					},
				},
			}

			sourceTerminalID, err = egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(sourceTerminalID).To(Equal(int64(1)))

			destinationTerminalID, err = egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(destinationTerminalID).To(Equal(int64(2)))

			egressPolicyID, err = egressPolicyTable.CreateEgressPolicy(tx, sourceTerminalID, destinationTerminalID)
			Expect(err).ToNot(HaveOccurred())
			Expect(egressPolicyID).To(Equal(int64(1)))

			appID, err = egressPolicyTable.CreateApp(tx, sourceTerminalID, "some-app-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(appID).To(Equal(int64(1)))

			ipRangeID, err = egressPolicyTable.CreateIPRange(tx, destinationTerminalID, "1.1.1.1", "2.2.2.2", "tcp", 8080, 8081, 0, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(ipRangeID).To(Equal(int64(1)))
		})

		It("should return all the ids for an egress policy", func() {
			ids, err := egressPolicyTable.GetIDCollectionsByEgressPolicy(tx, egressPolicy)
			Expect(err).NotTo(HaveOccurred())
			Expect(ids).To(Equal([]store.EgressPolicyIDCollection{{
				EgressPolicyID:        egressPolicyID,
				DestinationTerminalID: destinationTerminalID,
				DestinationIPRangeID:  ipRangeID,
				SourceTerminalID:      sourceTerminalID,
				SourceAppID:           appID,
				SourceSpaceID:         -1,
			}}))
		})

		Context("when there are duplicate matching policies", func() {
			var (
				destinationTerminalIDDuplicate int64
				egressPolicyIDDuplicate        int64
				ipRangeIDDuplicate             int64
			)

			BeforeEach(func() {
				var err error

				destinationTerminalIDDuplicate, err = egressPolicyTable.CreateTerminal(tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(destinationTerminalIDDuplicate).To(Equal(int64(3)))

				egressPolicyIDDuplicate, err = egressPolicyTable.CreateEgressPolicy(tx, sourceTerminalID, destinationTerminalIDDuplicate)
				Expect(err).ToNot(HaveOccurred())
				Expect(egressPolicyIDDuplicate).To(Equal(int64(2)))

				ipRangeIDDuplicate, err = egressPolicyTable.CreateIPRange(tx, destinationTerminalIDDuplicate, "1.1.1.1", "2.2.2.2", "tcp", 8080, 8081, 0, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(ipRangeIDDuplicate).To(Equal(int64(2)))
			})

			It("returns them all", func() {
				ids, err := egressPolicyTable.GetIDCollectionsByEgressPolicy(tx, egressPolicy)
				Expect(err).NotTo(HaveOccurred())
				Expect(ids).To(Equal([]store.EgressPolicyIDCollection{
					{
						EgressPolicyID:        egressPolicyID,
						DestinationTerminalID: destinationTerminalID,
						DestinationIPRangeID:  ipRangeID,
						SourceTerminalID:      sourceTerminalID,
						SourceAppID:           appID,
						SourceSpaceID:         -1,
					},
					{
						EgressPolicyID:        egressPolicyIDDuplicate,
						DestinationTerminalID: destinationTerminalIDDuplicate,
						DestinationIPRangeID:  ipRangeIDDuplicate,
						SourceTerminalID:      sourceTerminalID,
						SourceAppID:           appID,
						SourceSpaceID:         -1,
					},
				}))
			})
		})

		Context("when source terminal is attached to a space", func() {
			var (
				spaceSourceTerminalID int64
				spaceEgressPolicyID   int64
				spaceID               int64
				spaceEgressPolicy     store.EgressPolicy
			)

			BeforeEach(func() {
				spaceEgressPolicy = store.EgressPolicy{
					Source: store.EgressSource{
						ID:   "some-space-guid",
						Type: "space",
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
								Start: "1.1.1.1",
								End:   "2.2.2.2",
							},
						},
					},
				}

				var err error
				spaceSourceTerminalID, err = egressPolicyTable.CreateTerminal(tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(spaceSourceTerminalID).To(Equal(int64(3)))

				spaceID, err = egressPolicyTable.CreateSpace(tx, spaceSourceTerminalID, "some-space-guid")
				Expect(err).ToNot(HaveOccurred())
				Expect(spaceID).To(Equal(int64(1)))

				spaceEgressPolicyID, err = egressPolicyTable.CreateEgressPolicy(tx, spaceSourceTerminalID, destinationTerminalID)
				Expect(err).ToNot(HaveOccurred())
				Expect(spaceEgressPolicyID).To(Equal(int64(2)))
			})

			It("returns all the space id and sets app id to -1", func() {
				ids, err := egressPolicyTable.GetIDCollectionsByEgressPolicy(tx, spaceEgressPolicy)
				Expect(err).NotTo(HaveOccurred())
				Expect(ids).To(Equal([]store.EgressPolicyIDCollection{{
					EgressPolicyID:        spaceEgressPolicyID,
					DestinationTerminalID: destinationTerminalID,
					DestinationIPRangeID:  ipRangeID,
					SourceTerminalID:      spaceSourceTerminalID,
					SourceSpaceID:         spaceID,
					SourceAppID:           -1,
				}}))
			})
		})

		Context("when no port is provided", func() {
			BeforeEach(func() {
				var err error
				egressPolicy = store.EgressPolicy{
					Source: store.EgressSource{
						ID: "some-app-guid-2",
					},
					Destination: store.EgressDestination{
						Protocol: "tcp",
						IPRanges: []store.IPRange{
							{
								Start: "1.1.1.1",
								End:   "2.2.2.2",
							},
						},
					},
				}

				sourceTerminalID, err = egressPolicyTable.CreateTerminal(tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(sourceTerminalID).To(Equal(int64(3)))

				destinationTerminalID, err = egressPolicyTable.CreateTerminal(tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(destinationTerminalID).To(Equal(int64(4)))

				egressPolicyID, err = egressPolicyTable.CreateEgressPolicy(tx, sourceTerminalID, destinationTerminalID)
				Expect(err).ToNot(HaveOccurred())
				Expect(egressPolicyID).To(Equal(int64(2)))

				appID, err = egressPolicyTable.CreateApp(tx, sourceTerminalID, "some-app-guid-2")
				Expect(err).ToNot(HaveOccurred())
				Expect(appID).To(Equal(int64(2)))

				ipRangeID, err = egressPolicyTable.CreateIPRange(tx, destinationTerminalID, "1.1.1.1", "2.2.2.2", "tcp", 0, 0, 0, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(ipRangeID).To(Equal(int64(2)))
			})

			It("should returns the ids for the egress policy with port values of 0", func() {
				ids, err := egressPolicyTable.GetIDCollectionsByEgressPolicy(tx, egressPolicy)
				Expect(err).NotTo(HaveOccurred())
				Expect(ids).To(Equal([]store.EgressPolicyIDCollection{{
					EgressPolicyID:        egressPolicyID,
					DestinationTerminalID: destinationTerminalID,
					DestinationIPRangeID:  ipRangeID,
					SourceTerminalID:      sourceTerminalID,
					SourceAppID:           appID,
					SourceSpaceID:         -1,
				}}))
			})
		})

		Context("when icmp policy", func() {
			BeforeEach(func() {
				var err error
				egressPolicy = store.EgressPolicy{
					Source: store.EgressSource{
						ID: "some-app-guid-2",
					},
					Destination: store.EgressDestination{
						Protocol: "icmp",
						ICMPType: 1,
						ICMPCode: 2,
						IPRanges: []store.IPRange{
							{
								Start: "1.1.1.1",
								End:   "2.2.2.2",
							},
						},
					},
				}

				By("making a icmp policy")
				sourceTerminalID, err = egressPolicyTable.CreateTerminal(tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(sourceTerminalID).To(Equal(int64(3)))

				destinationTerminalID, err = egressPolicyTable.CreateTerminal(tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(destinationTerminalID).To(Equal(int64(4)))

				egressPolicyID, err = egressPolicyTable.CreateEgressPolicy(tx, sourceTerminalID, destinationTerminalID)
				Expect(err).ToNot(HaveOccurred())
				Expect(egressPolicyID).To(Equal(int64(2)))

				appID, err = egressPolicyTable.CreateApp(tx, sourceTerminalID, "some-app-guid-2")
				Expect(err).ToNot(HaveOccurred())
				Expect(appID).To(Equal(int64(2)))

				ipRangeID, err = egressPolicyTable.CreateIPRange(tx, destinationTerminalID, "1.1.1.1", "2.2.2.2", "icmp", 0, 0, 1, 2)
				Expect(err).ToNot(HaveOccurred())
				Expect(ipRangeID).To(Equal(int64(2)))

				By("making a slightly similar icmp policy with different type/code")
				otherDestTermID, err := egressPolicyTable.CreateTerminal(tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(otherDestTermID).To(Equal(int64(5)))

				otherEgressPolicyID, err := egressPolicyTable.CreateEgressPolicy(tx, sourceTerminalID, otherDestTermID)
				Expect(err).ToNot(HaveOccurred())
				Expect(otherEgressPolicyID).To(Equal(int64(3)))

				otherIpRangeID, err := egressPolicyTable.CreateIPRange(tx, otherDestTermID, "1.1.1.1", "2.2.2.2", "icmp", 0, 0, 3, 4)
				Expect(err).ToNot(HaveOccurred())
				Expect(otherIpRangeID).To(Equal(int64(3)))
			})

			It("should returns the ids for the egress policy that matches the icmp policy", func() {
				ids, err := egressPolicyTable.GetIDCollectionsByEgressPolicy(tx, egressPolicy)
				Expect(err).NotTo(HaveOccurred())
				Expect(ids).To(Equal([]store.EgressPolicyIDCollection{{
					EgressPolicyID:        egressPolicyID,
					DestinationTerminalID: destinationTerminalID,
					DestinationIPRangeID:  ipRangeID,
					SourceTerminalID:      sourceTerminalID,
					SourceAppID:           appID,
					SourceSpaceID:         -1,
				}}))
			})
		})

		Context("when it can't find a matching egress policy", func() {
			It("returns an empty collection", func() {
				otherEgressPolicy := store.EgressPolicy{
					Source: store.EgressSource{
						ID: "some-other-app-guid",
					},
					Destination: store.EgressDestination{
						Protocol: "udp",
						IPRanges: []store.IPRange{
							{
								Start: "1.1.1.1",
								End:   "2.2.2.2",
							},
						},
					},
				}
				results, err := egressPolicyTable.GetIDCollectionsByEgressPolicy(tx, otherEgressPolicy)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(0))
			})
		})
	})

	Context("GetTerminalByAppGUID", func() {
		It("should return the terminal id for an app if it exists", func() {
			terminalId, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())
			appsId, err := egressPolicyTable.CreateApp(tx, terminalId, "some-app-guid")
			Expect(err).ToNot(HaveOccurred())

			foundID, err := egressPolicyTable.GetTerminalByAppGUID(tx, "some-app-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(foundID).To(Equal(appsId))
		})

		It("should -1 and no error if the app is not found", func() {
			foundID, err := egressPolicyTable.GetTerminalByAppGUID(tx, "some-app-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(foundID).To(Equal(int64(-1)))
		})
	})

	Context("GetTerminalBySpaceGUID", func() {
		It("should return the terminal id for a space if it exists", func() {
			terminalId, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())
			spacesId, err := egressPolicyTable.CreateSpace(tx, terminalId, "some-space-guid")
			Expect(err).ToNot(HaveOccurred())

			foundID, err := egressPolicyTable.GetTerminalBySpaceGUID(tx, "some-space-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(foundID).To(Equal(spacesId))
		})

		It("should -1 and no error if the space is not found", func() {
			foundID, err := egressPolicyTable.GetTerminalBySpaceGUID(tx, "some-space-guid")
			Expect(err).ToNot(HaveOccurred())
			Expect(foundID).To(Equal(int64(-1)))
		})
	})

	Context("GetAllPolicies", func() {
		var egressPolicies []store.EgressPolicy

		BeforeEach(func() {
			egressPolicies = []store.EgressPolicy{
				{
					Source: store.EgressSource{
						ID:   "some-app-guid",
						Type: "app",
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
						ID:   "different-app-guid",
						Type: "app",
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
						ID:   "different-app-guid",
						Type: "app",
					},
					Destination: store.EgressDestination{
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
				},
				{
					Source: store.EgressSource{
						ID:   "space-guid",
						Type: "space",
					},
					Destination: store.EgressDestination{
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
				},
			}
			err := egressStore.CreateWithTx(tx, egressPolicies)
			Expect(err).ToNot(HaveOccurred())

			err = tx.Commit()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns policies", func() {
			listedPolicies, err := egressPolicyTable.GetAllPolicies()
			Expect(err).ToNot(HaveOccurred())
			Expect(listedPolicies).To(Equal(egressPolicies))
		})

		Context("when the query fails", func() {
			It("returns an error", func() {
				mockDb.QueryReturns(nil, errors.New("some error that sql would return"))

				egressPolicyTable = &store.EgressPolicyTable{
					Conn: mockDb,
				}

				_, err := egressPolicyTable.GetAllPolicies()
				Expect(err).To(MatchError("some error that sql would return"))
			})
		})
	})

	Context("GetByGuids", func() {
		var egressPolicies []store.EgressPolicy

		BeforeEach(func() {
			egressPolicies = []store.EgressPolicy{
				{
					Source: store.EgressSource{
						ID:   "some-app-guid",
						Type: "app",
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
						ID:   "different-app-guid",
						Type: "app",
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
						ID:   "different-app-guid",
						Type: "app",
					},
					Destination: store.EgressDestination{
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
				},
				{
					Source: store.EgressSource{
						ID:   "some-space-guid",
						Type: "space",
					},
					Destination: store.EgressDestination{
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
				},
				{
					Source: store.EgressSource{
						ID: "never-referenced-app-guid",
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
			}
			err := egressStore.CreateWithTx(tx, egressPolicies)
			Expect(err).ToNot(HaveOccurred())

			err = tx.Commit()
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when there are policies with the given id", func() {
			It("returns egress policies", func() {
				policies, err := egressPolicyTable.GetByGuids([]string{"some-app-guid", "different-app-guid", "some-space-guid"})
				Expect(err).ToNot(HaveOccurred())
				Expect(policies).To(ConsistOf(egressPolicies[:4]))
			})
		})

		Context("when there are no policies with the given id", func() {
			It("returns no egress policies", func() {
				policies, err := egressPolicyTable.GetByGuids([]string{"meow-this-is-a-bogus-app-guid"})
				Expect(err).ToNot(HaveOccurred())
				Expect(policies).To(Equal([]store.EgressPolicy{}))
			})
		})

		Context("when the query fails", func() {
			It("returns an error", func() {
				mockDb.QueryReturns(nil, errors.New("some error that sql would return"))

				egressPolicyTable = &store.EgressPolicyTable{
					Conn: mockDb,
				}

				_, err := egressPolicyTable.GetByGuids([]string{"id-does-not-matter"})
				Expect(err).To(MatchError("some error that sql would return"))
			})
		})

	})
})
