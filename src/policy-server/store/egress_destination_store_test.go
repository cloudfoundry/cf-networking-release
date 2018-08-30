package store_test

import (
	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"
	"errors"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"policy-server/db"
	dbfakes "policy-server/db/fakes"
	"policy-server/store"
	"policy-server/store/fakes"
	"policy-server/store/migrations"
	"strconv"
	"test-helpers"
	"time"
)

var _ = Describe("EgressDestinationStore", func() {
	var (
		egressDestinationsStore *store.EgressDestinationStore
		egressPolicyRepo        *store.EgressPolicyTable
		egressDestinationTable  *store.EgressDestinationTable
		mockDb                  *fakes.Db
		tx                      *dbfakes.Transaction
		dbConf                  dbHelper.Config
		realDb                  *db.ConnWrapper

		realMigrator *migrations.Migrator
	)

	BeforeEach(func() {
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("egress_destination_store_test_node_%d", time.Now().UnixNano())
		dbConf.Timeout = 30
		testhelpers.CreateDatabase(dbConf)

		logger := lager.NewLogger("Egress Destination Store Test")
		realDb = db.NewConnectionPool(dbConf, 200, 200, "Egress Destination Store Test", "Egress Destination Store Test", logger)
		mockDb = new(fakes.Db)
		tx = new(dbfakes.Transaction)

		mockDb.BeginxReturns(tx, nil)

		realMigrator = &migrations.Migrator{
			MigrateAdapter: &migrations.MigrateAdapter{},
			MigrationsProvider: &migrations.MigrationsProvider{
				Store: &store.MigrationsStore{
					DBConn: realDb,
				},
			},
		}

		_, err := realMigrator.PerformMigrations(realDb.DriverName(), realDb, 0)
		Expect(err).NotTo(HaveOccurred())

		egressDestinationTable = &store.EgressDestinationTable{}
		egressDestinationsStore = &store.EgressDestinationStore{
			Conn: realDb,
			EgressDestinationRepo: egressDestinationTable,
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

	Describe("All", func() {
		Context("when there are policies created", func() {
			var (
				destinationTerminalID1, destinationTerminalID2, destinationTerminalID3 int64
			)
			BeforeEach(func() {
				tx, err := realDb.Beginx()
				Expect(err).ToNot(HaveOccurred())
				destinationTerminalID1, err = egressPolicyRepo.CreateTerminal(tx)
				Expect(err).ToNot(HaveOccurred())
				destinationTerminalID2, err = egressPolicyRepo.CreateTerminal(tx)
				Expect(err).ToNot(HaveOccurred())
				destinationTerminalID3, err = egressPolicyRepo.CreateTerminal(tx)
				Expect(err).ToNot(HaveOccurred())
				Expect(tx.Commit()).ToNot(HaveOccurred())

				_, err = realDb.Exec(
					realDb.Rebind(`
					INSERT INTO ip_ranges (protocol, start_ip, end_ip, start_port, end_port, icmp_type, icmp_code, terminal_id)
					VALUES 
                    ('tcp', '1.2.2.2', '1.2.2.3', 8080, 8081, 9, 10, ?),
                    ('udp', '1.2.2.4', '1.2.2.5', 8083, 8087, 11, 14, ?),
                    ('icmp', '2.2.2.4', '2.2.2.5', 0, 0, 11, 14, ?);
				`), destinationTerminalID1, destinationTerminalID2, destinationTerminalID3)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return a list of all policies", func() {
				destinations, err := egressDestinationsStore.All()
				Expect(err).NotTo(HaveOccurred())
				Expect(destinations).To(Equal([]store.EgressDestination{
					{
						ID:       strconv.FormatInt(destinationTerminalID1, 10),
						Protocol: "tcp",
						IPRanges: []store.IPRange{{Start: "1.2.2.2", End: "1.2.2.3"}},
						Ports:    []store.Ports{{Start: 8080, End: 8081}},
						ICMPType: 9,
						ICMPCode: 10,
					},
					{
						ID:       strconv.FormatInt(destinationTerminalID2, 10),
						Protocol: "udp",
						IPRanges: []store.IPRange{{Start: "1.2.2.4", End: "1.2.2.5"}},
						Ports:    []store.Ports{{Start: 8083, End: 8087}},
						ICMPType: 11,
						ICMPCode: 14,
					},
					{
						ID:       strconv.FormatInt(destinationTerminalID3, 10),
						Protocol: "icmp",
						IPRanges: []store.IPRange{{Start: "2.2.2.4", End: "2.2.2.5"}},
						ICMPType: 11,
						ICMPCode: 14,
					},
				}))
			})

			Context("when the transaction cannot be created", func() {
				BeforeEach(func() {
					mockDb.BeginxReturns(nil, errors.New("can't create a transaction"))
					egressDestinationsStore.Conn = mockDb
				})

				It("returns an error when a transaction cannot be created", func() {
					_, err := egressDestinationsStore.All()
					Expect(err).To(MatchError("egress destination store create transaction: can't create a transaction"))
				})
			})

			Context("when the query fails", func() {
				BeforeEach(func() {
					mockDb.BeginxReturns(tx, nil)
					egressDestinationsStore.Conn = mockDb
					tx.QueryxReturns(nil, errors.New("query failed"))
				})

				It("returns an error when a transaction cannot be created", func() {
					_, err := egressDestinationsStore.All()
					Expect(err).To(MatchError("query failed"))
				})
			})
		})
	})
})
