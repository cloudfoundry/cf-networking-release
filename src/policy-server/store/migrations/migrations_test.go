package migrations

import (
	"database/sql"
	"fmt"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rubenv/sql-migrate"
)

type columnUsage struct {
	constraintName string
	columnName     string
}

var _ = Describe("migrations", func() {

	var dbConf db.Config
	var realDb *sqlx.DB

	BeforeEach(func() {
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("test_node_%d", GinkgoParallelNode())

		testsupport.CreateDatabase(dbConf)

		var err error
		realDb, err = db.GetConnectionPool(dbConf)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		testsupport.RemoveDatabase(dbConf)
	})

	Describe("V1", func() {

		Context("mysql", func() {
			BeforeEach(func() {
				if realDb.DriverName() != "mysql" {
					Skip("skipping mysql tests")
				}
			})

			It("should migrate", func() {
				numMigrations, err := PerformMigrations(realDb.DriverName(), realDb, &testMigrateAdapter{2})
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(2))

				rows, err := realDb.Query(`
				select CONSTRAINT_NAME, COLUMN_NAME
				from INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
				where TABLE_NAME='destinations'
			`)
				Expect(err).NotTo(HaveOccurred())

				By("checking there's a constraint on group_id, port, protocol")
				actualColumnUsageRows := scanColumnUsageRows(rows)

				Expect(actualColumnUsageRows).To(ConsistOf(columnUsage{
					constraintName: "PRIMARY",
					columnName:     "id",
				},
					columnUsage{
						constraintName: "group_id",
						columnName:     "group_id",
					},
					columnUsage{
						constraintName: "group_id",
						columnName:     "port",
					},
					columnUsage{
						constraintName: "group_id",
						columnName:     "protocol",
					},
				))
			})
		})

		Context("postgres", func() {
			BeforeEach(func() {
				if realDb.DriverName() != "postgres" {
					Skip("skipping postgres tests")
				}
			})

			It("should migrate", func() {
				numMigrations, err := PerformMigrations(realDb.DriverName(), realDb, &testMigrateAdapter{2})
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(2))

				rows, err := realDb.Query(`
				select CONSTRAINT_NAME, COLUMN_NAME
				from INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
				where TABLE_NAME='destinations'
			`)
				Expect(err).NotTo(HaveOccurred())

				By("checking there's a constraint on group_id, port, protocol")
				actualColumnUsageRows := scanColumnUsageRows(rows)
				Expect(actualColumnUsageRows).To(ConsistOf(columnUsage{
					constraintName: "destinations_pkey",
					columnName:     "id",
				},
					columnUsage{
						constraintName: "destinations_group_id_port_protocol_key",
						columnName:     "group_id",
					},
					columnUsage{
						constraintName: "destinations_group_id_port_protocol_key",
						columnName:     "port",
					},
					columnUsage{
						constraintName: "destinations_group_id_port_protocol_key",
						columnName:     "protocol",
					},
					columnUsage{
						constraintName: "destinations_group_id_fkey",
						columnName:     "group_id",
					},
				))
			})
		})
	})

	Describe("V2", func() {

		Context("mysql", func() {
			BeforeEach(func() {
				if realDb.DriverName() != "mysql" {
					Skip("skipping mysql tests")
				}
			})

			It("should migrate", func() {
				numMigrations, err := PerformMigrations(realDb.DriverName(), realDb, &testMigrateAdapter{3})
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(3))

				rows, err := realDb.Query(`
				select CONSTRAINT_NAME, COLUMN_NAME
				from INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
				where TABLE_NAME='destinations'
			`)
				Expect(err).NotTo(HaveOccurred())

				By("checking there's a constraint on group_id, port, protocol")
				actualColumnUsageRows := scanColumnUsageRows(rows)

				Expect(actualColumnUsageRows).To(ConsistOf(columnUsage{
					constraintName: "PRIMARY",
					columnName:     "id",
				},
					columnUsage{
						constraintName: "destinations_group_id_start_port_end_port_protocol_key",
						columnName:     "group_id",
					},
					columnUsage{
						constraintName: "destinations_group_id_start_port_end_port_protocol_key",
						columnName:     "start_port",
					},
					columnUsage{
						constraintName: "destinations_group_id_start_port_end_port_protocol_key",
						columnName:     "end_port",
					},
					columnUsage{
						constraintName: "destinations_group_id_start_port_end_port_protocol_key",
						columnName:     "protocol",
					},
				))
			})
		})

		Context("postgres", func() {
			BeforeEach(func() {
				if realDb.DriverName() != "postgres" {
					Skip("skipping postgres tests")
				}
			})

			It("should migrate", func() {
				numMigrations, err := PerformMigrations(realDb.DriverName(), realDb, &testMigrateAdapter{3})
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(3))

				rows, err := realDb.Query(`
				select CONSTRAINT_NAME, COLUMN_NAME
				from INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
				where TABLE_NAME='destinations'
			`)
				Expect(err).NotTo(HaveOccurred())

				By("checking there's a constraint on group_id, port, protocol")
				actualColumnUsageRows := scanColumnUsageRows(rows)
				Expect(actualColumnUsageRows).To(ConsistOf(columnUsage{
					constraintName: "destinations_pkey",
					columnName:     "id",
				},
					columnUsage{
						constraintName: "destinations_group_id_start_port_end_port_protocol_key",
						columnName:     "group_id",
					},
					columnUsage{
						constraintName: "destinations_group_id_start_port_end_port_protocol_key",
						columnName:     "start_port",
					},
					columnUsage{
						constraintName: "destinations_group_id_start_port_end_port_protocol_key",
						columnName:     "end_port",
					},
					columnUsage{
						constraintName: "destinations_group_id_start_port_end_port_protocol_key",
						columnName:     "protocol",
					},
					columnUsage{
						constraintName: "destinations_group_id_fkey",
						columnName:     "group_id",
					},
				))
			})
		})
	})

	//Describe("Migrations", func() {
	//	It("performs the migrations", func() {
	//		_, err := store.New(mockDb, mockMigrateAdapter, group, destination, policy, 1, 2*time.Second)
	//		Expect(err).NotTo(HaveOccurred())
	//
	//		By("calling the migrator")
	//		Expect(mockMigrateAdapter.ExecCallCount()).To(Equal(1))
	//		db, dbType, migrations, dir := mockMigrateAdapter.ExecArgsForCall(0)
	//		Expect(db).To(Equal(mockDb))
	//		Expect(dbType).To(Equal(mockDb.DriverName()))
	//
	//		Expect(migrations).To(Equal(migrate.MemoryMigrationSource{
	//			Migrations: []*migrate.Migration{
	//				{
	//					Id:   "1",
	//					Up:   store.Schemas[db.DriverName()],
	//					Down: []string{"DROP TABLE policies", "DROP TABLE destinations", "DROP TABLE groups"},
	//				},
	//				{
	//					Id:   "2",
	//					Up:   store.SchemasV1Up[db.DriverName()],
	//					Down: store.SchemasV1Down[db.DriverName()],
	//				},
	//			},
	//		}))
	//
	//		Expect(dir).To(Equal(migrate.Up))
	//	})
	//
	//	Context("when the driver name is not mysql or postgres", func() {
	//		BeforeEach(func() {
	//			mockDb.DriverNameReturns("etcd")
	//		})
	//		It("returns an error", func() {
	//			_, err := store.New(mockDb, mockMigrateAdapter, group, destination, policy, 1, 2*time.Second)
	//			Expect(err).To(MatchError("setting up tables: unsupported driver: etcd"))
	//		})
	//	})
	//
	//	Context("when the migrations fail", func() {
	//		BeforeEach(func() {
	//			mockMigrateAdapter.ExecReturns(0, errors.New("banana"))
	//		})
	//		It("returns an error", func() {
	//			_, err := store.New(mockDb, mockMigrateAdapter, group, destination, policy, 1, 2*time.Second)
	//			Expect(err).To(MatchError("setting up tables: executing migration: banana"))
	//		})
	//	})
	//})
})

func scanColumnUsageRows(rows *sql.Rows) []columnUsage {
	actual := []columnUsage{}
	for rows.Next() {
		var constraintName string
		var columnName string

		Expect(rows.Scan(&constraintName, &columnName)).To(Succeed())
		actual = append(actual, columnUsage{
			constraintName: constraintName,
			columnName:     columnName,
		})
	}
	return actual
}

type testMigrateAdapter struct {
	migrateUpTo int
}

func (tma testMigrateAdapter) Exec(db MigrationDb, dialect string, migrationSource migrate.MigrationSource, dir migrate.MigrationDirection) (int, error) {
	allMigrations, err := migrationSource.FindMigrations()
	Expect(err).ToNot(HaveOccurred())

	newMemoryMigrationSource := migrate.MemoryMigrationSource{
		Migrations: allMigrations[:tma.migrateUpTo],
	}

	return migrate.Exec(db.(*sqlx.DB).DB, dialect, newMemoryMigrationSource, dir)
}
