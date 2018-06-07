package migrations_test

import (
	"database/sql"
	"errors"
	"fmt"
	"policy-server/store/fakes"
	"policy-server/store/helpers"
	"policy-server/store/migrations"
	migrationsFakes "policy-server/store/migrations/fakes"

	"sync"

	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"github.com/cf-container-networking/sql-migrate"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type columnUsage struct {
	value      string
	columnName string
}

var _ = Describe("migrations", func() {

	var (
		dbConf             db.Config
		realDb             *sqlx.DB
		mockDb             *fakes.Db
		realMigrateAdapter *migrations.MigrateAdapter
		mockMigrateAdapter *migrationsFakes.MigrateAdapter
		migrator           *migrations.Migrator
	)

	BeforeEach(func() {
		mockDb = &fakes.Db{}
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("migrator_test_node_%d", time.Now().UnixNano())
		dbConf.Timeout = 30
		testsupport.CreateDatabase(dbConf)

		var err error
		realDb, err = db.GetConnectionPool(dbConf)
		Expect(err).NotTo(HaveOccurred())

		realMigrateAdapter = &migrations.MigrateAdapter{}
		mockMigrateAdapter = &migrationsFakes.MigrateAdapter{}

		migrator = &migrations.Migrator{
			MigrateAdapter: &migrations.MigrateAdapter{},
		}
	})

	AfterEach(func() {
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		testsupport.RemoveDatabase(dbConf)
	})

	Describe("PerformMigrations", func() {
		Describe("V1", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}
				})

				It("should migrate", func() {
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(1))

					By("checking the destinations, groups, and policies tables were created")

					By("checking there's a constraint on group_id, port, protocol", func() {
						rows, err := realDb.Query(helpers.RebindForSQLDialect(`
							select CONSTRAINT_NAME, COLUMN_NAME
							from INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
							where TABLE_NAME='destinations' and TABLE_SCHEMA=?
						`, realDb.DriverName()), dbConf.DatabaseName)

						Expect(err).NotTo(HaveOccurred())
						actualColumnUsageRows := scanColumnUsageRows(rows)

						Expect(actualColumnUsageRows).To(ConsistOf(
							columnUsage{value: "PRIMARY", columnName: "id"},
							columnUsage{value: "group_id", columnName: "group_id"},
							columnUsage{value: "group_id", columnName: "port"},
							columnUsage{value: "group_id", columnName: "protocol"},
						))
					})
				})
			})
			Context("postgres", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "postgres" {
						Skip("skipping postgres tests")
					}
				})

				It("should migrate", func() {
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(1))

					By("checking there's a constraint on group_id, port, protocol", func() {
						rows, err := realDb.Query(`
							select CONSTRAINT_NAME, COLUMN_NAME
							from INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
							where TABLE_NAME='destinations'
						`)
						Expect(err).NotTo(HaveOccurred())

						actualColumnUsageRows := scanColumnUsageRows(rows)
						Expect(actualColumnUsageRows).To(ConsistOf(
							columnUsage{
								value:      "destinations_pkey",
								columnName: "id",
							},
							columnUsage{
								value:      "destinations_group_id_port_protocol_key",
								columnName: "group_id",
							},
							columnUsage{
								value:      "destinations_group_id_port_protocol_key",
								columnName: "port",
							},
							columnUsage{
								value:      "destinations_group_id_port_protocol_key",
								columnName: "protocol",
							},
							columnUsage{
								value:      "destinations_group_id_fkey",
								columnName: "group_id",
							},
						))
					})
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
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 2)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(2))

					rows, err := realDb.Query(helpers.RebindForSQLDialect(`
							select CONSTRAINT_NAME, COLUMN_NAME
							from INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
							where TABLE_NAME='destinations' and TABLE_SCHEMA=?
						`, realDb.DriverName()), dbConf.DatabaseName)
					Expect(err).NotTo(HaveOccurred())

					By("checking there's a constraint on group_id, start_port, end_port, protocol")
					actualColumnUsageRows := scanColumnUsageRows(rows)

					Expect(actualColumnUsageRows).To(ConsistOf(
						columnUsage{
							value:      "PRIMARY",
							columnName: "id",
						},
						columnUsage{
							value:      "unique_destination",
							columnName: "group_id",
						},
						columnUsage{
							value:      "unique_destination",
							columnName: "start_port",
						},
						columnUsage{
							value:      "unique_destination",
							columnName: "end_port",
						},
						columnUsage{
							value:      "unique_destination",
							columnName: "protocol",
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
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 2)
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
						value:      "destinations_pkey",
						columnName: "id",
					},
						columnUsage{
							value:      "unique_destination",
							columnName: "group_id",
						},
						columnUsage{
							value:      "unique_destination",
							columnName: "start_port",
						},
						columnUsage{
							value:      "unique_destination",
							columnName: "end_port",
						},
						columnUsage{
							value:      "unique_destination",
							columnName: "protocol",
						},
						columnUsage{
							value:      "destinations_group_id_fkey",
							columnName: "group_id",
						},
					))
				})
			})
		})

		Describe("V3", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}
				})

				It("should migrate", func() {
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 2) //v1, v2
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(2))

					By("inserting existing data")
					_, err = realDb.Exec(`insert into groups (guid) values ("some-guid")`)
					Expect(err).NotTo(HaveOccurred())

					By("performing migration")
					numMigrations, err = migrator.PerformMigrations(realDb.DriverName(), realDb, 1) //v3
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(1))

					By("verifying existing rows have type 'app'")
					rows, err := realDb.Query(`
							SELECT count(*)
							FROM groups
							WHERE type = 'app' AND guid = 'some-guid'
						`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))

					By("inserting new data")
					_, err = realDb.Exec(`insert into groups (guid) values ("some-new-guid")`)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row defaults to type 'app'")
					rows, err = realDb.Query(`
							SELECT count(*)
							FROM groups
							WHERE type = 'app' AND guid = 'some-new-guid'
						`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))

					By("inserting new data with a type")
					_, err = realDb.Exec(`insert into groups (guid, type) values ("some-new-guid-router", "router")`)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row has correct type")
					rows, err = realDb.Query(`
							SELECT count(*)
							FROM groups
							WHERE type = 'router' AND guid = 'some-new-guid-router'
					`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))
				})

				It("has an index on the group.type column", func() {
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 3)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(3))

					rows, err := realDb.Query(`
							SELECT DISTINCT INDEX_NAME, COLUMN_NAME
							FROM INFORMATION_SCHEMA.STATISTICS
							WHERE TABLE_NAME='groups'
					`)
					Expect(err).NotTo(HaveOccurred())

					By("checking there's an index")
					actualColumnUsageRows := scanColumnUsageRows(rows)
					Expect(actualColumnUsageRows).To(ConsistOf(
						columnUsage{columnName:"id", value: "PRIMARY"},
						columnUsage{columnName:"guid", value: "guid"},
						columnUsage{columnName:"type", value: "idx_type"},
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
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 2) //v1, v2
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(2))

					By("inserting existing data")
					_, err = realDb.Query(`INSERT INTO groups (guid) VALUES ('some-guid')`) // must be single quotes
					Expect(err).NotTo(HaveOccurred())

					By("performing migration")
					numMigrations, err = migrator.PerformMigrations(realDb.DriverName(), realDb, 1) //v3
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(1))

					By("verifying existing rows have type 'app'")
					rows, err := realDb.Query(`
							SELECT count(*)
							FROM groups
							WHERE type = 'app' AND guid = 'some-guid'
						`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))

					By("inserting new data")
					_, err = realDb.Exec(`insert into groups (guid) values ('some-new-guid')`)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row defaults to type 'app'")
					rows, err = realDb.Query(`
							SELECT count(*)
							FROM groups
							WHERE type = 'app' AND guid = 'some-new-guid'
						`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))

					By("inserting new data with a type")
					_, err = realDb.Exec(`insert into groups (guid, type) values ('some-new-guid-router', 'router')`)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row has correct type")
					rows, err = realDb.Query(`
							SELECT count(*)
							FROM groups
							WHERE type = 'router' AND guid = 'some-new-guid-router'
					`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))
				})

				It("has an index on the group.type column", func() {
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 3)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(3))

					rows, err := realDb.Query(`
						SELECT indexdef, indexname FROM pg_indexes WHERE tablename = 'groups'
					`)
					Expect(err).NotTo(HaveOccurred())


					By("checking there's an index")
					actualColumnUsageRows := scanColumnUsageRows(rows)
					Expect(actualColumnUsageRows).To(ConsistOf(
						columnUsage{columnName:"groups_pkey", value: "CREATE UNIQUE INDEX groups_pkey ON groups USING btree (id)"},
						columnUsage{columnName:"groups_guid_key", value: "CREATE UNIQUE INDEX groups_guid_key ON groups USING btree (guid)"},
						columnUsage{columnName:"idx_type", value: "CREATE INDEX idx_type ON groups USING btree (type)"},
					))
				})
			})
		})

		Context("when migrating in parallel", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}
				})

				It("should migrate", func() {
					numOfRoutines := 10
					wg := sync.WaitGroup{}
					wg.Add(numOfRoutines)

					for i := 0; i < numOfRoutines; i++ {
						go func() {
							defer wg.Done()
							defer GinkgoRecover()

							_, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 0)
							Expect(err).ToNot(HaveOccurred())
						}()
					}

					wg.Wait()
				}, 10)
			})

			Context("postgres", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "postgres" {
						Skip("skipping postgres tests")
					}
				})

				It("should migrate", func() {
					numOfRoutines := 10
					wg := sync.WaitGroup{}
					wg.Add(numOfRoutines)

					for i := 0; i < numOfRoutines; i++ {
						go func() {
							defer wg.Done()
							defer GinkgoRecover()

							_, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 0)
							Expect(err).ToNot(HaveOccurred())
						}()
					}

					wg.Wait()
				}, 10)
			})

		})

		Context("when the driver name is not mysql or postgres", func() {
			It("returns an error", func() {
				_, err := migrator.PerformMigrations("etcd", mockDb, 2)
				Expect(err).To(MatchError("unsupported driver: etcd"))
			})
		})

		Context("when the migrations fail", func() {
			BeforeEach(func() {
				migrator.MigrateAdapter = mockMigrateAdapter
				mockMigrateAdapter.ExecMaxReturns(0, errors.New("banana"))
			})
			It("returns an error", func() {
				_, err := migrator.PerformMigrations(realDb.DriverName(), mockDb, 2)
				Expect(err).To(MatchError("executing migration: banana"))
				Expect(mockMigrateAdapter.ExecMaxCallCount()).To(Equal(1))
				db, driverName, _, migrationDir, numMigrations := mockMigrateAdapter.ExecMaxArgsForCall(0)
				Expect(db).To(Equal(mockDb))
				Expect(driverName).To(Equal(realDb.DriverName()))
				Expect(migrationDir).To(Equal(migrate.Up))
				Expect(numMigrations).To(Equal(2))
			})
		})
	})

	Describe("Down Migration", func() {
		It("should no-op", func() {
			adapter := &migrations.MigrateAdapter{}

			_, err := adapter.ExecMax(
				realDb,
				realDb.DriverName(),
				migrate.MemoryMigrationSource{
					Migrations: migrations.MigrationsToPerform.ForDriver(realDb.DriverName()),
				},
				migrate.Up,
				0,
			)
			Expect(err).NotTo(HaveOccurred())

			numberOfMigrations, err := adapter.ExecMax(
				realDb,
				realDb.DriverName(),
				migrate.MemoryMigrationSource{
					Migrations: migrations.MigrationsToPerform.ForDriver(realDb.DriverName()),
				},
				migrate.Down,
				0,
			)

			Expect(err).To(MatchError("down migration not supported"))
			Expect(numberOfMigrations).To(Equal(0))
		})
	})
})

func scanColumnUsageRows(rows *sql.Rows) []columnUsage {
	actual := []columnUsage{}
	defer rows.Close()
	for rows.Next() {
		var constraintName string
		var columnName string

		Expect(rows.Scan(&constraintName, &columnName)).To(Succeed())
		actual = append(actual, columnUsage{
			value:      constraintName,
			columnName: columnName,
		})
	}
	Expect(rows.Err()).NotTo(HaveOccurred())
	return actual
}

func scanCountRow(rows *sql.Rows) int {
	defer rows.Close()
	count := 0
	for rows.Next() {
		Expect(rows.Scan(&count)).To(Succeed())
	}
	return count
}
