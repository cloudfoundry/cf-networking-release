package migrations_test

import (
	"database/sql"
	"errors"
	"fmt"
	"policy-server/store"
	"policy-server/store/fakes"
	"policy-server/store/helpers"
	"policy-server/store/migrations"
	migrationsFakes "policy-server/store/migrations/fakes"

	"sync"

	"time"

	"policy-server/db"
	"test-helpers"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"
	"github.com/cf-container-networking/sql-migrate"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type columnUsage struct {
	value      string
	columnName string
}

var _ = Describe("migrations", func() {

	var (
		dbConf                     dbHelper.Config
		realDb                     *db.ConnWrapper
		mockDb                     *fakes.Db
		mockMigrateAdapter         *migrationsFakes.MigrateAdapter
		legacyMigrations           migrations.PolicyServerMigrations
		legacyMigrationsProvider   *migrationsFakes.MigrationsProvider
		modifiedMigrationsProvider *migrations.MigrationsProvider
		legacyMigrator             *migrations.Migrator
		migrator                   *migrations.Migrator
	)

	BeforeEach(func() {
		mockDb = &fakes.Db{}
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("migrator_test_node_%d", time.Now().UnixNano())
		dbConf.Timeout = 30
		testhelpers.CreateDatabase(dbConf)

		logger := lager.NewLogger("Migrations Test")

		realDb = db.NewConnectionPool(dbConf, 200, 200, "Store Test", "Store Test", logger)

		mockMigrateAdapter = &migrationsFakes.MigrateAdapter{}

		legacyMigrations = append(
			migrations.V1LegacyMigrationsToPerform,
			migrations.V2LegacyMigrationsToPerform[0],
			migrations.V2LegacyMigrationsToPerform[1], //a
			migrations.V2LegacyMigrationsToPerform[2], //b
			migrations.V2LegacyMigrationsToPerform[3], //c
			migrations.V2LegacyMigrationsToPerform[4], //d
			migrations.V2LegacyMigrationsToPerform[5], //e
			migrations.V2LegacyMigrationsToPerform[6], //f
			migrations.V3LegacyMigrationsToPerform[0],
			migrations.V3LegacyMigrationsToPerform[1], //a
		)
		legacyMigrations = append(legacyMigrations,
			migrations.MigrationsToPerform...)

		legacyMigrationsProvider = &migrationsFakes.MigrationsProvider{}
		legacyMigrationsProvider.MigrationsToPerformReturns(legacyMigrations, nil)
		legacyMigrator = &migrations.Migrator{
			MigrateAdapter:     &migrations.MigrateAdapter{},
			MigrationsProvider: legacyMigrationsProvider,
		}

		modifiedMigrationsProvider = &migrations.MigrationsProvider{
			Store: &store.MigrationsStore{
				DBConn: realDb,
			},
		}

		migrator = &migrations.Migrator{
			MigrateAdapter:     &migrations.MigrateAdapter{},
			MigrationsProvider: modifiedMigrationsProvider,
		}
	})

	AfterEach(func() {
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		testhelpers.RemoveDatabase(dbConf)
	})

	Describe("PerformMigrations", func() {
		Describe("V1", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}
				})

				It("should migrate 1, 1a, 1b", func() {
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 3)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(3))

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

				Context("when legacy migration v1 has already run", func() {
					BeforeEach(func() {
						numMigrations, err := legacyMigrator.PerformMigrations(realDb.DriverName(), realDb, 1)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(1))
					})

					It("should migrate with empty 1a, 1b", func() {
						numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 2)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(2))

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

						By("checking the gorp_migrations table for 1b and 1c", func() {
							expectMigrations(realDb, []string{"1", "1a", "1b"})
						})
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
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 3)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(3))

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

				Context("when legacy migration v1 has already run", func() {
					BeforeEach(func() {
						numMigrations, err := legacyMigrator.PerformMigrations(realDb.DriverName(), realDb, 1)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(1))
					})

					It("should migrate with empty 1a, 1b", func() {
						numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 2)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(2))

						By("checking the destinations, groups, and policies tables were created")

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

						By("checking the gorp_migrations table for 1b and 1c", func() {
							expectMigrations(realDb, []string{"1", "1a", "1b"})
						})
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
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 10)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(10))

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

				Context("when legacy migration v2 has already run", func() {
					BeforeEach(func() {
						numMigrations, err := legacyMigrator.PerformMigrations(realDb.DriverName(), realDb, 4)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(4))
					})

					It("should migrate with empty 2a-2f", func() {
						numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 6)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(6))

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

						By("checking the gorp_migrations table for 2a-2f", func() {
							expectMigrations(realDb, []string{"1", "1a", "1b", "2", "2a", "2b", "2c", "2d", "2e", "2f"})
						})
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
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 10)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(10))

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

				Context("when legacy migration v2 has already run", func() {
					BeforeEach(func() {
						numToRun := len(migrations.V1LegacyMigrationsToPerform) + 1
						numMigrations, err := legacyMigrator.PerformMigrations(realDb.DriverName(), realDb, numToRun)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(numToRun))
					})

					It("should migrate with empty 2a-2f", func() {
						numToRun := len(migrations.V2ModifiedMigrationsToPerform) - 1
						numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, numToRun)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(numToRun))

						rows, err := realDb.Query(`
							select CONSTRAINT_NAME, COLUMN_NAME
							from INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
							where TABLE_NAME='destinations'
						`)
						Expect(err).NotTo(HaveOccurred())
						defer rows.Close()

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

						By("checking the gorp_migrations table for 2a-2f", func() {
							expectMigrations(realDb, []string{"1", "1a", "1b", "2", "2a", "2b", "2c", "2d", "2e", "2f"})
						})
					})
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
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 10) //v1, v2
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(10))

					By("inserting existing data")
					_, err = realDb.Exec(`insert into groups (guid) values ("some-guid")`)
					Expect(err).NotTo(HaveOccurred())

					By("performing migration")
					numMigrations, err = migrator.PerformMigrations(realDb.DriverName(), realDb, 2) //v3
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(2))

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
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 12) //v1, v2, v3
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(12))

					rows, err := realDb.Query(`
							SELECT DISTINCT INDEX_NAME, COLUMN_NAME
							FROM INFORMATION_SCHEMA.STATISTICS
							WHERE TABLE_NAME='groups'
					`)
					Expect(err).NotTo(HaveOccurred())

					By("checking there's an index")
					actualColumnUsageRows := scanColumnUsageRows(rows)
					Expect(actualColumnUsageRows).To(ConsistOf(
						columnUsage{columnName: "id", value: "PRIMARY"},
						columnUsage{columnName: "guid", value: "guid"},
						columnUsage{columnName: "type", value: "idx_type"},
					))
				})

				Context("when legacy migration v3 has already run", func() {
					BeforeEach(func() {
						numMigrations, err := legacyMigrator.PerformMigrations(realDb.DriverName(), realDb, 11)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(11))
					})

					It("should migrate with empty 3a", func() {
						numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(1))

						rows, err := realDb.Query(`
							SELECT DISTINCT INDEX_NAME, COLUMN_NAME
							FROM INFORMATION_SCHEMA.STATISTICS
							WHERE TABLE_NAME='groups'
						`)
						Expect(err).NotTo(HaveOccurred())

						By("checking there's an index")
						actualColumnUsageRows := scanColumnUsageRows(rows)
						Expect(actualColumnUsageRows).To(ConsistOf(
							columnUsage{columnName: "id", value: "PRIMARY"},
							columnUsage{columnName: "guid", value: "guid"},
							columnUsage{columnName: "type", value: "idx_type"},
						))

						By("checking the gorp_migrations table for 3a", func() {
							expectMigrations(realDb, []string{"1", "1a", "1b", "2", "2a", "2b", "2c", "2d", "2e", "2f", "3", "3a"})
						})
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
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 10) //v1, v2
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(10))

					By("inserting existing data")
					_, err = realDb.Exec(`insert into groups (guid) values ('some-guid')`)
					Expect(err).NotTo(HaveOccurred())

					By("performing migration")
					numMigrations, err = migrator.PerformMigrations(realDb.DriverName(), realDb, 2) //v3
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(2))

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
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 12)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(12))

					rows, err := realDb.Query(`
						SELECT indexdef, indexname FROM pg_indexes WHERE tablename = 'groups'
					`)
					Expect(err).NotTo(HaveOccurred())

					By("checking there's an index")
					actualColumnUsageRows := scanColumnUsageRows(rows)
					Expect(actualColumnUsageRows).To(ConsistOf(
						columnUsage{columnName: "groups_pkey", value: "CREATE UNIQUE INDEX groups_pkey ON public.groups USING btree (id)"},
						columnUsage{columnName: "groups_guid_key", value: "CREATE UNIQUE INDEX groups_guid_key ON public.groups USING btree (guid)"},
						columnUsage{columnName: "idx_type", value: "CREATE INDEX idx_type ON public.groups USING btree (type)"},
					))
				})

				Context("when legacy migration v3 has already run", func() {
					BeforeEach(func() {
						numMigrations, err := legacyMigrator.PerformMigrations(realDb.DriverName(), realDb, 11)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(11))
					})

					It("should migrate with empty 3a", func() {
						numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
						Expect(err).NotTo(HaveOccurred())
						Expect(numMigrations).To(Equal(1))

						rows, err := realDb.Query(`
						SELECT indexdef, indexname FROM pg_indexes WHERE tablename = 'groups'
						`)
						Expect(err).NotTo(HaveOccurred())

						By("checking there's an index")
						actualColumnUsageRows := scanColumnUsageRows(rows)
						Expect(actualColumnUsageRows).To(ConsistOf(
							columnUsage{columnName: "groups_pkey", value: "CREATE UNIQUE INDEX groups_pkey ON public.groups USING btree (id)"},
							columnUsage{columnName: "groups_guid_key", value: "CREATE UNIQUE INDEX groups_guid_key ON public.groups USING btree (guid)"},
							columnUsage{columnName: "idx_type", value: "CREATE INDEX idx_type ON public.groups USING btree (type)"},
						))

						By("checking the gorp_migrations table for 3a", func() {
							expectMigrations(realDb, []string{"1", "1a", "1b", "2", "2a", "2b", "2c", "2d", "2e", "2f", "3", "3a"})
						})
					})
				})
			})
		})

		Describe("V4", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}
				})

				It("should migrate", func() {
					By("performing migration")
					numMigrationsPerformed, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 13) //v1, v2, v3, v4
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrationsPerformed).To(Equal(13))

					By("verifying there are no rows")
					rows, err := realDb.Query(`
							SELECT count(*)
							FROM terminals
						`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(0))

					By("inserting new data")
					_, err = realDb.Exec(`insert into terminals (id) values (NULL)`)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row exists")
					rows, err = realDb.Query(`
							SELECT count(*)
							FROM terminals
						`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))

				})
			})

			Context("postgres", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "postgres" {
						Skip("skipping postgres tests")
					}
				})

				It("should migrate", func() {
					numMigrationsPerformed, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 13) //v1, v2, v3, v4
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrationsPerformed).To(Equal(13))

					By("verifying there are no rows")
					rows, err := realDb.Query(`
							SELECT count(*)
							FROM terminals
						`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(0))

					By("inserting new data")
					_, err = realDb.Exec(`insert into terminals default values`)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row exists")
					rows, err = realDb.Query(`
							SELECT count(*)
							FROM terminals
						`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))
				})
			})
		})

		Describe("V5", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}

					By("performing migration")
					numMigrationsPerformed, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 14) //v1, v2, v3, v4
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrationsPerformed).To(Equal(14))
				})

				It("should migrate", func() {
					By("verifying there are no rows")
					rows, err := realDb.Query(`SELECT count(*) FROM egress_policies`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(0))

					result, err := realDb.Exec("INSERT INTO terminals (id) VALUES (NULL)")
					Expect(err).NotTo(HaveOccurred())
					terminalId, err := result.LastInsertId()
					Expect(err).NotTo(HaveOccurred())

					By("inserting new data")
					_, err = realDb.Exec(`
						INSERT INTO egress_policies (source_id, destination_id)
						VALUES (?, ?)`, terminalId, terminalId)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row exists")
					rows, err = realDb.Query(`
						SELECT id FROM egress_policies
						WHERE source_id = 1 AND destination_id = 1`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))
				})

				It("constrains the terminal id to existing rows", func() {
					_, err := realDb.Exec(`
						INSERT INTO egress_policies (source_id, destination_id)
						VALUES (42, 23)`)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("foreign key constraint fails"))
				})
			})

			Context("postgres", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "postgres" {
						Skip("skipping postgres tests")
					}

					By("performing migration")
					numMigrationsPerformed, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 14) //v1, v2, v3, v4
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrationsPerformed).To(Equal(14))
				})

				It("should migrate", func() {
					By("verifying there are no rows")
					rows, err := realDb.Query(`SELECT count(*) FROM egress_policies`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(0))

					By("creating a policy to associate to")
					var terminalId int64
					err = realDb.QueryRow("INSERT INTO terminals default values RETURNING id").Scan(&terminalId)
					Expect(err).NotTo(HaveOccurred())

					By("inserting new data")
					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO egress_policies (source_id, destination_id) 
						VALUES (?, ?)`), terminalId, terminalId)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row exists")
					rows, err = realDb.Query(`
						SELECT id FROM egress_policies 
						WHERE source_id=1 AND destination_id=1`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))
				})

				It("constrains the terminal id to existing rows", func() {
					_, err := realDb.Exec(`
						INSERT INTO egress_policies (source_id, destination_id) 
						VALUES (42, 23)`)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("violates foreign key constraint"))
				})
			})
		})

		Describe("V6", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}

					By("performing migration")
					numMigrationsPerformed, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 15) //v1, v2, v3, v4
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrationsPerformed).To(Equal(15))
				})

				It("should migrate", func() {
					By("verifying there are no rows")
					rows, err := realDb.Query(`SELECT count(*) FROM ip_ranges`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(0))

					result, err := realDb.Exec("INSERT INTO terminals (id) VALUES (NULL)")
					Expect(err).NotTo(HaveOccurred())
					terminalId, err := result.LastInsertId()
					Expect(err).NotTo(HaveOccurred())

					By("inserting new data")
					_, err = realDb.Exec(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id) 
						VALUES ('tcp', '1.2.3.4', '2.3.4.5', ?)`, terminalId)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row exists")
					rows, err = realDb.Query(`
						SELECT id FROM ip_ranges 
						WHERE protocol='tcp' AND start_ip='1.2.3.4' AND end_ip='2.3.4.5' AND terminal_id=1`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))
				})

				It("constrains the policy id to existing rows", func() {
					_, err := realDb.Exec(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id) 
						VALUES ('tcp', '1.2.3.4', '2.3.4.5', 42)`)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("foreign key constraint fails"))
				})
			})

			Context("postgres", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "postgres" {
						Skip("skipping postgres tests")
					}

					By("performing migration")
					numMigrationsPerformed, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 15) //v1, v2, v3, v4
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrationsPerformed).To(Equal(15))
				})

				It("should migrate", func() {
					By("verifying there are no rows")
					rows, err := realDb.Query(`SELECT count(*) FROM ip_ranges`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(0))

					By("creating a policy to associate to")
					var terminalId int64
					err = realDb.QueryRow("INSERT INTO terminals default values RETURNING id").Scan(&terminalId)
					Expect(err).NotTo(HaveOccurred())

					By("inserting new data")
					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id) 
						VALUES ('tcp', '1.2.3.4', '2.3.4.5', ?)`), terminalId)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row exists")
					rows, err = realDb.Query(`
						SELECT id FROM ip_ranges 
						WHERE protocol='tcp' AND start_ip='1.2.3.4' AND end_ip='2.3.4.5' AND terminal_id=1`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))
				})

				It("constrains the policy id to existing rows", func() {
					_, err := realDb.Exec(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id) 
						VALUES ('tcp','1.2.3.4','2.3.4.5',42)`)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("violates foreign key constraint"))
				})
			})
		})

		Describe("V7", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}

					By("performing migration")
					numMigrationsPerformed, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 16) //v1, v2, v3, v4
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrationsPerformed).To(Equal(16))
				})

				It("should migrate", func() {
					By("verifying there are no rows")
					rows, err := realDb.Query(`SELECT count(*) FROM apps`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(0))

					By("inserting a required endpoint")
					result, err := realDb.Exec("INSERT INTO terminals (id) VALUES (NULL)")
					Expect(err).NotTo(HaveOccurred())
					terminalId, err := result.LastInsertId()
					Expect(err).NotTo(HaveOccurred())

					By("inserting new data")
					_, err = realDb.Exec(`INSERT INTO apps (terminal_id, app_guid) VALUES (?,'an-app-guid')`, terminalId)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row exists")
					rows, err = realDb.Query(realDb.RawConnection().Rebind(`SELECT id FROM apps WHERE id=1 AND terminal_id=? AND app_guid='an-app-guid'`), terminalId)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))
				})

				It("constrains the terminal id to existing rows", func() {
					_, err := realDb.Exec(`INSERT INTO apps (terminal_id, app_guid) VALUES (42,'an-app-guid')`)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("foreign key constraint fails"))
				})
			})

			Context("postgres", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "postgres" {
						Skip("skipping postgres tests")
					}

					By("performing migration")
					numMigrationsPerformed, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 16) //v1, v2, v3, v4
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrationsPerformed).To(Equal(16))
				})

				It("should migrate", func() {
					By("verifying there are no rows")
					rows, err := realDb.Query(`SELECT count(*) FROM apps`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(0))

					By("creating ab endpoint to associate to")
					var terminalId int64
					err = realDb.QueryRow("INSERT INTO terminals DEFAULT VALUES RETURNING id").Scan(&terminalId)
					Expect(err).NotTo(HaveOccurred())

					By("inserting new data")
					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
							INSERT INTO apps (terminal_id, app_guid) 
							VALUES (?,'an-app-guid')`), terminalId)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row exists")
					rows, err = realDb.Query(realDb.RawConnection().Rebind(`SELECT id FROM apps WHERE id=1 AND terminal_id=? AND app_guid='an-app-guid'`), terminalId)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))
				})

				It("constrains the terminal id to existing rows", func() {
					_, err := realDb.Exec(`INSERT INTO apps (terminal_id, app_guid) VALUES (42,'an-app-guid')`)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("violates foreign key constraint"))
				})
			})
		})

		Describe("V8-11 PG Indexes", func() {
			Context("postgres", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "postgres" {
						Skip("skipping mysql tests")
					}

					numMigrationsPerformed, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 20) //v1, v2, v3, v4
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrationsPerformed).To(Equal(20))
				})

				It("should have indexes on foreign keys", func() {
					rows, err := realDb.Query(`SELECT tablename, indexname FROM pg_indexes WHERE tablename = 'egress_policies'`)
					Expect(err).NotTo(HaveOccurred())
					actualColumnUsageRows := scanColumnUsageRows(rows)

					Expect(actualColumnUsageRows).To(ConsistOf(
						columnUsage{value: "egress_policies", columnName: "egress_policies_pkey"},
						columnUsage{value: "egress_policies", columnName: "source_terminal_id_idx"},
						columnUsage{value: "egress_policies", columnName: "destination_terminal_id_idx"},
					))

					rows, err = realDb.Query(`SELECT tablename, indexname FROM pg_indexes WHERE tablename = 'ip_ranges'`)
					Expect(err).NotTo(HaveOccurred())
					actualColumnUsageRows = scanColumnUsageRows(rows)

					Expect(actualColumnUsageRows).To(ConsistOf(
						columnUsage{value: "ip_ranges", columnName: "ip_ranges_pkey"},
						columnUsage{value: "ip_ranges", columnName: "ip_range_terminal_id_idx"},
					))

					rows, err = realDb.Query(`SELECT tablename, indexname FROM pg_indexes WHERE tablename = 'apps'`)
					Expect(err).NotTo(HaveOccurred())
					actualColumnUsageRows = scanColumnUsageRows(rows)

					Expect(actualColumnUsageRows).To(ConsistOf(
						columnUsage{value: "apps", columnName: "apps_pkey"},
						columnUsage{value: "apps", columnName: "apps_app_guid_unique"},
						columnUsage{value: "apps", columnName: "app_terminal_id_idx"},
					))
				})
			})
		})

		Describe("V12-V15 IP Range Ports", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}

					By("performing migration")
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 20)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(20))

					By("inserting data")
					result, err := realDb.Exec("INSERT INTO terminals (id) VALUES (NULL)")
					Expect(err).NotTo(HaveOccurred())
					terminalId, err := result.LastInsertId()
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id)
						VALUES (?, ?, ?, ?)`, "tcp", "1.2.3.4", "1.2.3.5", terminalId)
					Expect(err).NotTo(HaveOccurred())

					By("performing migration for ip range ports")
					numMigrations, err = migrator.PerformMigrations(realDb.DriverName(), realDb, 4)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(4))
				})

				It("should migrate", func() {
					rows, err := realDb.Query(helpers.RebindForSQLDialect(`
							select COLUMN_NAME
							from INFORMATION_SCHEMA.COLUMNS t1
							where TABLE_NAME='ip_ranges' and TABLE_SCHEMA=?
						`, realDb.DriverName()), dbConf.DatabaseName)
					Expect(err).NotTo(HaveOccurred())

					By("verifying the start and end port columns exist", func() {
						var columns []string
						defer rows.Close()
						for rows.Next() {
							var columnName string
							Expect(rows.Scan(&columnName)).To(Succeed())
							columns = append(columns, columnName)
						}
						Expect(columns).To(ContainElement("start_port"))
						Expect(columns).To(ContainElement("end_port"))
					})

					By("verifying that old rows have a default value of 0 for start/end ports", func() {
						var startPort, endPort int64
						err := realDb.QueryRow(`SELECT start_port, end_port FROM ip_ranges`).Scan(&startPort, &endPort)
						Expect(err).NotTo(HaveOccurred())
						Expect(startPort).To(Equal(int64(0)))
						Expect(startPort).To(Equal(int64(0)))
					})
				})
			})

			Context("postgres", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "postgres" {
						Skip("skipping postgres tests")
					}

					By("performing migration")
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 20)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(20))

					By("inserting data")
					var terminalId int64
					err = realDb.QueryRow("INSERT INTO terminals DEFAULT VALUES RETURNING id").Scan(&terminalId)
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id)
						VALUES (?, ?, ?, ?)`), "tcp", "1.2.3.4", "1.2.3.5", terminalId)
					Expect(err).NotTo(HaveOccurred())

					By("performing migration for ip range ports")
					numMigrations, err = migrator.PerformMigrations(realDb.DriverName(), realDb, 4)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(4))
				})

				It("should migrate", func() {
					rows, err := realDb.Query(helpers.RebindForSQLDialect(`
							select COLUMN_NAME
							from INFORMATION_SCHEMA.COLUMNS t1
							where TABLE_NAME='ip_ranges'
						`, realDb.DriverName()))
					Expect(err).NotTo(HaveOccurred())

					By("verifying the start and end port columns exist", func() {
						var columns []string
						defer rows.Close()
						for rows.Next() {
							var columnName string
							Expect(rows.Scan(&columnName)).To(Succeed())
							columns = append(columns, columnName)
						}
						Expect(columns).To(ContainElement("start_port"))
						Expect(columns).To(ContainElement("end_port"))
					})

					By("verifying that old rows have a default value of 0 for start/end ports", func() {
						var startPort, endPort int64
						err := realDb.QueryRow(`SELECT start_port, end_port FROM ip_ranges`).Scan(&startPort, &endPort)
						Expect(err).NotTo(HaveOccurred())
						Expect(startPort).To(Equal(int64(0)))
						Expect(startPort).To(Equal(int64(0)))
					})
				})
			})
		})

		Describe("V16-V18 ICMP Range", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}

					By("performing migration")
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 24)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(24))

					By("inserting data")
					result, err := realDb.Exec("INSERT INTO terminals (id) VALUES (NULL)")
					Expect(err).NotTo(HaveOccurred())
					terminalId, err := result.LastInsertId()
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id, start_port, end_port)
						VALUES (?, ?, ?, ?, ?, ?)`, "tcp", "1.2.3.4", "1.2.3.5", terminalId, 8080, 8081)
					Expect(err).NotTo(HaveOccurred())

					By("performing migration for icmp type/code")
					numMigrations, err = migrator.PerformMigrations(realDb.DriverName(), realDb, 2)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(2))
				})

				It("should migrate", func() {
					rows, err := realDb.Query(helpers.RebindForSQLDialect(`
							select COLUMN_NAME
							from INFORMATION_SCHEMA.COLUMNS t1
							where TABLE_NAME='ip_ranges' and TABLE_SCHEMA=?
						`, realDb.DriverName()), dbConf.DatabaseName)
					Expect(err).NotTo(HaveOccurred())

					By("verifying the icmp type and icmp code columns exist", func() {
						var columns []string
						defer rows.Close()
						for rows.Next() {
							var columnName string
							Expect(rows.Scan(&columnName)).To(Succeed())
							columns = append(columns, columnName)
						}
						Expect(columns).To(ContainElement("icmp_type"))
						Expect(columns).To(ContainElement("icmp_code"))
					})

					By("verifying that old rows have a default value of 0 for start/end ports", func() {
						var icmpType, icmpCode int64
						err := realDb.QueryRow(`SELECT icmp_type, icmp_code FROM ip_ranges`).Scan(&icmpType, &icmpCode)
						Expect(err).NotTo(HaveOccurred())
						Expect(icmpType).To(Equal(int64(0)))
						Expect(icmpCode).To(Equal(int64(0)))
					})
				})
			})

			Context("postgres", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "postgres" {
						Skip("skipping postgres tests")
					}

					By("performing migration")
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 24)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(24))

					By("inserting data")
					var terminalId int64
					err = realDb.QueryRow("INSERT INTO terminals DEFAULT VALUES RETURNING id").Scan(&terminalId)
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id, start_port, end_port)
						VALUES (?, ?, ?, ?, ?, ?)`), "tcp", "1.2.3.4", "1.2.3.5", terminalId, 8080, 8081)
					Expect(err).NotTo(HaveOccurred())

					By("performing migration for icmp type/code")
					numMigrations, err = migrator.PerformMigrations(realDb.DriverName(), realDb, 2)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(2))
				})

				It("should migrate", func() {
					rows, err := realDb.Query(helpers.RebindForSQLDialect(`
							select COLUMN_NAME
							from INFORMATION_SCHEMA.COLUMNS t1
							where TABLE_NAME='ip_ranges'
						`, realDb.DriverName()))
					Expect(err).NotTo(HaveOccurred())

					By("verifying the icmp type and icmp code columns exist", func() {
						var columns []string
						defer rows.Close()
						for rows.Next() {
							var columnName string
							Expect(rows.Scan(&columnName)).To(Succeed())
							columns = append(columns, columnName)
						}
						Expect(columns).To(ContainElement("icmp_type"))
						Expect(columns).To(ContainElement("icmp_code"))
					})

					By("verifying that old rows have a default value of 0 for icmp type/code", func() {
						var icmpType, icmpCode int64
						err := realDb.QueryRow(`SELECT icmp_type, icmp_code FROM ip_ranges`).Scan(&icmpType, &icmpCode)
						Expect(err).NotTo(HaveOccurred())
						Expect(icmpType).To(Equal(int64(0)))
						Expect(icmpCode).To(Equal(int64(0)))
					})
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

		Context("when getting migrations to perform fails", func() {
			It("returns a meaningful error message", func() {
				legacyMigrationsProvider.MigrationsToPerformReturns(nil, errors.New("mark mark mark"))
				_, err := legacyMigrator.PerformMigrations(realDb.DriverName(), realDb, 0)
				Expect(err).To(MatchError("error retrieving migrations to perform: mark mark mark"))
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

func expectMigrations(realDb *db.ConnWrapper, expectedMigrations []string) {
	rows, err := realDb.Query(`select ID from gorp_migrations`)
	defer rows.Close()
	Expect(err).NotTo(HaveOccurred())
	var actual []string
	for rows.Next() {
		var id string

		Expect(rows.Scan(&id)).To(Succeed())
		actual = append(actual, id)
	}
	Expect(rows.Err()).NotTo(HaveOccurred())
	Expect(actual).To(Equal(expectedMigrations))
}

func scanColumnUsageRows(rows *sql.Rows) []columnUsage {
	var actual []columnUsage
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
