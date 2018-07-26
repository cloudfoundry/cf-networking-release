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

	Context("CreateIPRange", func() {
		It("should create an iprange and return the ID", func() {
			ipRangeTerminalID, err := egressPolicyTable.CreateTerminal(tx)
			Expect(err).ToNot(HaveOccurred())

			id, err := egressPolicyTable.CreateIPRange(tx, ipRangeTerminalID, "1.1.1.1", "2.2.2.2", "tcp")
			Expect(err).ToNot(HaveOccurred())

			Expect(id).To(Equal(int64(1)))

			var startIP, endIP, protocol string
			row := tx.QueryRow(`SELECT start_ip, end_ip, protocol FROM ip_ranges WHERE id = 1`)
			err = row.Scan(&startIP, &endIP, &protocol)
			Expect(err).ToNot(HaveOccurred())
			Expect(startIP).To(Equal("1.1.1.1"))
			Expect(endIP).To(Equal("2.2.2.2"))
			Expect(protocol).To(Equal("tcp"))
		})

		It("should return an error if the driver is not supported", func() {
			fakeTx := &dbfakes.Transaction{}

			fakeTx.DriverNameReturns("db2")

			_, err := egressPolicyTable.CreateIPRange(fakeTx, 1, "1.1.1.1", "2.2.2.2", "tcp")
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

			ipRangeID, err = egressPolicyTable.CreateIPRange(tx, ipRangeTerminalID, "1.1.1.1", "2.2.2.2", "tcp")
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

	Context("GetIDsByEgressPolicy", func() {
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

			ipRangeID, err = egressPolicyTable.CreateIPRange(tx, destinationTerminalID, "1.1.1.1", "2.2.2.2", "tcp")
			Expect(err).ToNot(HaveOccurred())
			Expect(ipRangeID).To(Equal(int64(1)))
		})

		It("should return all the ids for an egress policy", func() {
			ids, err := egressPolicyTable.GetIDsByEgressPolicy(tx, egressPolicy)
			Expect(err).NotTo(HaveOccurred())
			Expect(ids).To(Equal(store.EgressPolicyIDCollection{
				EgressPolicyID:        egressPolicyID,
				DestinationTerminalID: destinationTerminalID,
				DestinationIPRangeID:  ipRangeID,
				SourceTerminalID:      sourceTerminalID,
				SourceAppID:           appID,
			}))
		})

		Context("when it can't find a matching egress policy", func() {
			It("returns an error", func() {
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
				_, err := egressPolicyTable.GetIDsByEgressPolicy(tx, otherEgressPolicy)
				Expect(err).To(MatchError("sql: no rows in result set"))
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

	Context("GetAllPolicies", func() {
		var egressPolicies []store.EgressPolicy

		BeforeEach(func() {
			egressPolicies = []store.EgressPolicy{
				{
					Source: store.EgressSource{
						ID: "some-app-guid",
					},
					Destination: store.EgressDestination{
						Protocol: "tcp",
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
						ID: "some-app-guid",
					},
					Destination: store.EgressDestination{
						Protocol: "tcp",
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
				policies, err := egressPolicyTable.GetByGuids([]string{"some-app-guid", "different-app-guid"})
				Expect(err).ToNot(HaveOccurred())
				Expect(policies).To(Equal(egressPolicies[:2]))
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
