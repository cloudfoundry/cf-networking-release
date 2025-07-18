package migrations_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/policy-server/store"
	"code.cloudfoundry.org/policy-server/store/fakes"
	"code.cloudfoundry.org/policy-server/store/helpers"
	"code.cloudfoundry.org/policy-server/store/migrations"
	migrationsFakes "code.cloudfoundry.org/policy-server/store/migrations/fakes"
	testhelpers "code.cloudfoundry.org/test-helpers"
	uuid "github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	migrate "github.com/rubenv/sql-migrate"
)

type columnUsage struct {
	value      string
	columnName string
}

var _ = Describe("migrations", func() {

	var (
		dbConf                     db.Config
		realDb                     *db.ConnWrapper
		mockDb                     *fakes.Db
		mockMigrateAdapter         *migrationsFakes.MigrateAdapter
		legacyMigrations           migrations.PolicyServerMigrations
		legacyMigrationsProvider   *migrationsFakes.MigrationsProvider
		modifiedMigrationsProvider *migrations.MigrationsProvider
		legacyMigrator             *migrations.Migrator
		migrator                   *migrations.Migrator
	)

	var previousMigrationId string
	migrateTo := func(migrationId string) {
		var steps int
		if previousMigrationId != "" {
			By(fmt.Sprintf("migrating from %s to %s", previousMigrationId, migrationId))
			migrationIdx := getMigrationIndex(modifiedMigrationsProvider, migrationId)
			previousMigrationIdx := getMigrationIndex(modifiedMigrationsProvider, previousMigrationId)
			steps = migrationIdx - previousMigrationIdx
		} else {
			By("migrating to " + migrationId)
			steps = getMigrationIndex(modifiedMigrationsProvider, migrationId)
		}
		numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, steps)
		Expect(err).NotTo(HaveOccurred())
		Expect(numMigrations).To(Equal(steps))
		previousMigrationId = migrationId
	}

	BeforeEach(func() {
		previousMigrationId = ""
		mockDb = &fakes.Db{}
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("migrator_test_node_%d", time.Now().UnixNano())
		dbConf.Timeout = 30
		testhelpers.CreateDatabase(dbConf)

		logger := lager.NewLogger("Migrations Test")

		var err error
		realDb, err = db.NewConnectionPool(dbConf, 200, 0, 60*time.Minute, "Store Test", "Store Test", logger)
		Expect(err).NotTo(HaveOccurred())

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
					migrateTo("1b")

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
					migrateTo("1b")

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
					migrateTo("2f")

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
					migrateTo("2f")

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
					migrateTo("3a")

					By("inserting existing data")
					_, err := realDb.Exec(`insert into "groups" (guid) values ('some-guid')`)
					Expect(err).NotTo(HaveOccurred())

					By("performing migration")
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 2) //v3
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(2))

					By("verifying existing rows have type 'app'")
					rows, err := realDb.Query(`
							SELECT count(*)
							FROM "groups"
							WHERE type = 'app' AND guid = 'some-guid'
						`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))

					By("inserting new data")
					_, err = realDb.Exec(`insert into "groups" (guid) values ('some-new-guid')`)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row defaults to type 'app'")
					rows, err = realDb.Query(`
							SELECT count(*)
							FROM "groups"
							WHERE type = 'app' AND guid = 'some-new-guid'
						`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))

					By("inserting new data with a type")
					_, err = realDb.Exec(`insert into "groups" (guid, type) values ('some-new-guid-router', 'router')`)
					Expect(err).NotTo(HaveOccurred())

					By("verifying new row has correct type")
					rows, err = realDb.Query(`
							SELECT count(*)
							FROM "groups"
							WHERE type = 'router' AND guid = 'some-new-guid-router'
					`)
					Expect(err).NotTo(HaveOccurred())
					Expect(scanCountRow(rows)).To(Equal(1))
				})

				It("has an index on the group.type column", func() {
					migrateTo("3a")

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
					migrateTo("3a")

					By("inserting existing data")
					_, err := realDb.Exec(`insert into groups (guid) values ('some-guid')`)
					Expect(err).NotTo(HaveOccurred())

					By("performing migration")
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 2) //v3
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
					migrateTo("3a")

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
					migrateTo("4")

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
					migrateTo("4")

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
					migrateTo("5")
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
					migrateTo("5")
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
					migrateTo("6")
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
					migrateTo("6")
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
					migrateTo("7")
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
					migrateTo("7")
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
						Skip("skipping postgres tests")
					}
				})

				It("should have indexes on foreign keys", func() {
					migrateTo("11")

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
					migrateTo("11")

					By("inserting data")
					result, err := realDb.Exec("INSERT INTO terminals (id) VALUES (NULL)")
					Expect(err).NotTo(HaveOccurred())
					terminalId, err := result.LastInsertId()
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id)
						VALUES (?, ?, ?, ?)`, "tcp", "1.2.3.4", "1.2.3.5", terminalId)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should migrate", func() {
					By("performing migration for ip range ports")
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 4)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(4))

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
					migrateTo("11")

					By("inserting data")
					var terminalId int64
					err := realDb.QueryRow("INSERT INTO terminals DEFAULT VALUES RETURNING id").Scan(&terminalId)
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id)
						VALUES (?, ?, ?, ?)`), "tcp", "1.2.3.4", "1.2.3.5", terminalId)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should migrate", func() {
					By("performing migration for ip range ports")
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 4)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(4))

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

		Describe("V16-V17 ICMP Range", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}

					By("performing migration")
					migrateTo("15")

					By("inserting data")
					result, err := realDb.Exec("INSERT INTO terminals (id) VALUES (NULL)")
					Expect(err).NotTo(HaveOccurred())
					terminalId, err := result.LastInsertId()
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id, start_port, end_port)
						VALUES (?, ?, ?, ?, ?, ?)`, "tcp", "1.2.3.4", "1.2.3.5", terminalId, 8080, 8081)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should migrate", func() {
					By("performing migration for icmp type/code")
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 2)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(2))

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
					migrateTo("15")

					By("inserting data")
					var terminalId int64
					err := realDb.QueryRow("INSERT INTO terminals DEFAULT VALUES RETURNING id").Scan(&terminalId)
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id, start_port, end_port)
						VALUES (?, ?, ?, ?, ?, ?)`), "tcp", "1.2.3.4", "1.2.3.5", terminalId, 8080, 8081)
					Expect(err).NotTo(HaveOccurred())

					By("performing migration for icmp type/code")
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 2)
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

		Describe("V18 - spaces", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}
				})

				It("should migrate", func() {
					By("performing migration")
					migrateTo("18")

					rows, err := realDb.Query(helpers.RebindForSQLDialect(`
							select COLUMN_NAME
							from INFORMATION_SCHEMA.COLUMNS t1
							where TABLE_NAME='spaces'
						`, realDb.DriverName()))
					Expect(err).NotTo(HaveOccurred())

					By("verifying the terminal_id column exists", func() {
						var columns []string
						defer rows.Close()
						for rows.Next() {
							var columnName string
							Expect(rows.Scan(&columnName)).To(Succeed())
							columns = append(columns, columnName)
						}
						Expect(columns).To(ContainElement("terminal_id"))
						Expect(columns).To(ContainElement("space_guid"))
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
					By("performing migration")
					migrateTo("18")

					rows, err := realDb.Query(helpers.RebindForSQLDialect(`
							select COLUMN_NAME
							from INFORMATION_SCHEMA.COLUMNS t1
							where TABLE_NAME='spaces'
						`, realDb.DriverName()))
					Expect(err).NotTo(HaveOccurred())

					By("verifying the terminal_id column exists", func() {
						var columns []string
						defer rows.Close()
						for rows.Next() {
							var columnName string
							Expect(rows.Scan(&columnName)).To(Succeed())
							columns = append(columns, columnName)
						}
						Expect(columns).To(ContainElement("terminal_id"))
						Expect(columns).To(ContainElement("space_guid"))
					})
				})
			})
		})

		Describe("V19 through 21 - Named Destinations", func() {
			Context("mysql", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "mysql" {
						Skip("skipping mysql tests")
					}
				})

				It("should migrate", func() {
					By("performing migration")

					migrateTo("21")

					result, err := realDb.Exec("INSERT INTO terminals (id) VALUES (NULL)")
					Expect(err).NotTo(HaveOccurred())
					terminalId, err := result.LastInsertId()
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO destination_metadatas (terminal_id, name, description)
						VALUES (?, ?, ?)`), terminalId, "some-dest", "my destination")
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("postgres", func() {
				BeforeEach(func() {
					if realDb.DriverName() != "postgres" {
						Skip("skipping postgres tests")
					}
				})

				It("should migrate", func() {
					By("performing migration")
					migrateTo("21")

					var terminalId int64
					err := realDb.QueryRow("INSERT INTO terminals DEFAULT VALUES RETURNING id").Scan(&terminalId)
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO destination_metadatas (terminal_id, name, description)
						VALUES (?, ?, ?)`), terminalId, "some-dest", "my destination")
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		Describe("V22 through 50 - ID to GUID Named Destination", func() {
			Context("mysql", func() {
				var (
					terminalId int64
				)

				BeforeEach(func() {
					By("performing migration")
					migrateTo("21")

					terminalId = insertTerminal(realDb)

					_, err := realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO apps (terminal_id, app_guid)
						VALUES (?, ?)`), terminalId, "some-app-guid")
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO spaces (terminal_id, space_guid)
						VALUES (?, ?)`), terminalId, "some-space-guid")
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_id, start_port, end_port, icmp_type, icmp_code)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?)`), "tcp", "1.1.1.1", "2.2.2.2", terminalId, 8080, 8081, -1, -1)
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO destination_metadatas (terminal_id, name, description)
						VALUES (?, ?, ?)`), terminalId, "some-name", "some-description")
					Expect(err).NotTo(HaveOccurred())

					_, err = realDb.Exec(realDb.RawConnection().Rebind(`
						INSERT INTO egress_policies (source_id, destination_id)
						VALUES (?, ?)`), terminalId, terminalId)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should migrate", func() {
					By("performing migration")
					numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 29 /* there are 29 migrations required! */)
					Expect(err).NotTo(HaveOccurred())
					Expect(numMigrations).To(Equal(29))

					By("verifying the id was migrated to guid")
					expectedTermainalGUID := strconv.FormatInt(terminalId, 10)
					terminalGUIDs := queryTableForColumnValues("terminals", "guid", realDb)
					Expect(terminalGUIDs).To(ConsistOf(expectedTermainalGUID))

					expectedTermainalGUID = strconv.FormatInt(terminalId, 10)
					terminalGUIDs = queryTableForColumnValues("apps", "terminal_guid", realDb)
					Expect(terminalGUIDs).To(ConsistOf(expectedTermainalGUID))

					terminalGUIDs = queryTableForColumnValues("spaces", "terminal_guid", realDb)
					Expect(terminalGUIDs).To(ConsistOf(expectedTermainalGUID))

					terminalGUIDs = queryTableForColumnValues("ip_ranges", "terminal_guid", realDb)
					Expect(terminalGUIDs).To(ConsistOf(expectedTermainalGUID))

					terminalGUIDs = queryTableForColumnValues("destination_metadatas", "terminal_guid", realDb)
					Expect(terminalGUIDs).To(ConsistOf(expectedTermainalGUID))

					terminalGUIDs = queryTableForColumnValues("egress_policies", "source_guid", realDb)
					Expect(terminalGUIDs).To(ConsistOf(expectedTermainalGUID))

					terminalGUIDs = queryTableForColumnValues("egress_policies", "destination_guid", realDb)
					Expect(terminalGUIDs).To(ConsistOf(expectedTermainalGUID))

					By("verifying the terminal_guid column exists and the terminal_id column does not")
					Expect(queryTableColumnNames("terminals", realDb)).NotTo(ContainElement("id"))
					Expect(queryTableColumnNames("apps", realDb)).NotTo(ContainElement("terminal_id"))
					Expect(queryTableColumnNames("spaces", realDb)).NotTo(ContainElement("terminal_id"))
					Expect(queryTableColumnNames("ip_ranges", realDb)).NotTo(ContainElement("terminal_id"))
					Expect(queryTableColumnNames("destination_metadatas", realDb)).NotTo(ContainElement("terminal_id"))
					Expect(queryTableColumnNames("egress_policies", realDb)).NotTo(ContainElement("source_id"))
					Expect(queryTableColumnNames("egress_policies", realDb)).NotTo(ContainElement("destination_id"))
				})
			})
		})

		Describe("V51 through V55 - GUID Egress Policy", func() {
			BeforeEach(func() {
				By("performing migration")
				migrateTo("50")

				_, err := realDb.Exec("INSERT INTO terminals (guid) VALUES ('some-guid')")
				Expect(err).NotTo(HaveOccurred())

				_, err = realDb.Exec(realDb.RawConnection().Rebind(`
					INSERT INTO egress_policies (source_guid, destination_guid)
					VALUES (?, ?)`), "some-guid", "some-guid")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should migrate", func() {
				By("performing migration")
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 5 /* it takes 5 steps to get here */)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(5))

				By("verifying the guid column exists and the id column does not")
				Expect(queryTableColumnNames("egress_policies", realDb)).To(ContainElement("guid"))
				Expect(queryTableColumnNames("egress_policies", realDb)).NotTo(ContainElement("id"))

				By("verifying that, for old rows, the guid is just the numeric id")
				guid := queryTableForColumnValues("egress_policies", "guid", realDb)
				Expect(guid).To(ConsistOf("1"))

			})
		})

		Describe("V56 - Egress Policy uniqueness constraint", func() {
			It("should migrate", func() {
				By("performing migration")
				migrateTo("56")

				_, err := realDb.Exec("INSERT INTO terminals (guid) VALUES ('some-terminal-guid')")
				Expect(err).NotTo(HaveOccurred())

				By("validating that inserting the same policy twice fails")
				_, err = realDb.Exec(realDb.RawConnection().Rebind(`
					INSERT INTO egress_policies (guid, source_guid, destination_guid)
					VALUES (?, ?, ?)`), "some-egress-guid-1", "some-terminal-guid", "some-terminal-guid")
				Expect(err).NotTo(HaveOccurred())

				_, err = realDb.Exec(realDb.RawConnection().Rebind(`
					INSERT INTO egress_policies (guid, source_guid, destination_guid)
					VALUES (?, ?, ?)`), "some-egress-guid-2", "some-terminal-guid", "some-terminal-guid")
				Expect(err).To(MatchError(Or(
					ContainSubstring("duplicate key value violates unique constraint"), // postgres error
					ContainSubstring("Duplicate entry"),                                // mysql error
				)))
			})
		})

		Describe("V57 - V59 - Add app_lifecycle to egress_policies", func() {
			It("should migrate", func() {
				By("performing migration")
				migrateTo("56")
				_, err := realDb.Exec("INSERT INTO terminals (guid) VALUES ('some-terminal-guid')")
				Expect(err).NotTo(HaveOccurred())

				_, err = realDb.Exec("INSERT INTO terminals (guid) VALUES ('some-terminal-guid-also')")
				Expect(err).NotTo(HaveOccurred())

				By("inserting a policy before the migration")
				_, err = realDb.Exec(realDb.RawConnection().Rebind(`
					INSERT INTO egress_policies (guid, source_guid, destination_guid)
					VALUES (?, ?, ?)`), "some-egress-guid-1", "some-terminal-guid", "some-terminal-guid")
				Expect(err).NotTo(HaveOccurred())

				By("performing migration")
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 3)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(3))

				By("verifying old row uses default")
				var appLifecycle string
				err = realDb.QueryRow(`
						SELECT app_lifecycle FROM egress_policies
						WHERE guid = 'some-egress-guid-1'`).Scan(&appLifecycle)
				Expect(err).NotTo(HaveOccurred())
				Expect(appLifecycle).To(Equal("all"))

				By("validating that value can be inserted")
				_, err = realDb.Exec(realDb.RawConnection().Rebind(`
					INSERT INTO egress_policies (guid, source_guid, destination_guid, app_lifecycle)
					VALUES (?, ?, ?, ?)`), "some-egress-guid-2", "some-terminal-guid-also", "some-terminal-guid", "running")
				Expect(err).NotTo(HaveOccurred())

				By("validating that app lifecycle is considered in uniqueness constraint")
				_, err = realDb.Exec(realDb.RawConnection().Rebind(`
					INSERT INTO egress_policies (guid, source_guid, destination_guid, app_lifecycle)
					VALUES (?, ?, ?, ?)`), "some-egress-guid-3", "some-terminal-guid-also", "some-terminal-guid", "staging")
				Expect(err).NotTo(HaveOccurred())

				By("validating that it doesn't use default when provided")
				err = realDb.QueryRow(`
						SELECT app_lifecycle FROM egress_policies
						WHERE guid = 'some-egress-guid-2'`).Scan(&appLifecycle)
				Expect(err).NotTo(HaveOccurred())
				Expect(appLifecycle).To(Equal("running"))
			})
		})

		Describe("V60 - Add default table", func() {
			It("should migrate", func() {
				By("performing migration")
				migrateTo("60")

				_, err := realDb.Exec("INSERT INTO terminals (guid) VALUES ('some-terminal-guid')")
				Expect(err).NotTo(HaveOccurred())

				By("inserting a default")
				_, err = realDb.Exec(realDb.RawConnection().Rebind(`
					INSERT INTO defaults (terminal_guid)
					VALUES (?)`), "some-terminal-guid")
				Expect(err).NotTo(HaveOccurred())

				By("validating that it is inserted with an id")
				var id int
				var terminalGUID string
				err = realDb.QueryRow(`
						SELECT id, terminal_guid FROM defaults
						WHERE terminal_guid = 'some-terminal-guid'`).Scan(&id, &terminalGUID)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(1))
				Expect(terminalGUID).To(Equal("some-terminal-guid"))

				By("validating that terminal guid is unique")
				_, err = realDb.Exec(realDb.RawConnection().Rebind(`
					INSERT INTO defaults (terminal_guid)
					VALUES (?)`), "some-terminal-guid")
				Expect(err).To(MatchError(Or(
					ContainSubstring("duplicate key value violates unique constraint"), // postgres error
					ContainSubstring("Duplicate entry"),                                // mysql error
				)))

				By("validating that terminal guid is a foreign key")
				_, err = realDb.Exec(realDb.RawConnection().Rebind(`
					INSERT INTO defaults (terminal_guid)
					VALUES (?)`), "non-existant-terminal-guid")
				Expect(err).To(MatchError(ContainSubstring("foreign key constraint")))
			})
		})

		Describe("V61-V63 - Allow many ip ranges to an egress policy", func() {
			It("should migrate", func() {
				By("performing migration")
				migrateTo("60")

				_, err := realDb.Exec("INSERT INTO terminals (guid) VALUES ('some-terminal-guid')")
				Expect(err).NotTo(HaveOccurred())

				By("performing migration")
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 3)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(3))

				By("inserting two ip range for a terminal")
				_, err = realDb.Exec(`
					INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_guid)
					VALUES ('tcp', '1.2.3.4', '2.3.4.5', 'some-terminal-guid')`)
				Expect(err).NotTo(HaveOccurred())

				By("validating a duplicate ip range can be added")
				_, err = realDb.Exec(`
					INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_guid)
					VALUES ('tcp', '1.2.3.4', '2.3.4.5', 'some-terminal-guid')`)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("V64 - Add description to rules", func() {
			It("should add the column", func() {
				By("migrating up to 64")
				migrateTo("64")

				By("inserting a friendly ip range")
				_, err := realDb.Exec("INSERT INTO terminals (guid) VALUES ('some-terminal-guid')")
				Expect(err).NotTo(HaveOccurred())

				By("seeing that we create a ip_range with a description")
				_, err = realDb.Exec(`
					INSERT INTO ip_ranges (protocol, start_ip, end_ip, terminal_guid, description)
					VALUES ('udp', '2.2.2.2', '2.2.2.2', 'some-terminal-guid', 'it works!')`)
				Expect(err).NotTo(HaveOccurred())

				var description string
				err = realDb.QueryRow(`
						SELECT description FROM ip_ranges
						WHERE start_ip='2.2.2.2'`).Scan(&description)
				Expect(err).NotTo(HaveOccurred())
				Expect(description).To(Equal("it works!"))
			})
		})

		Describe("V65 - Prevent policies without destinations", func() {
			It("should have foreign key constraints after the migration", func() {
				By("migrating up to 65")
				migrateTo("65")

				var srcGroupId, dstGroupId, dstId, policyId int64

				By("inserting a src group")
				_, err := realDb.Exec(`insert into "groups" (guid) values ('src-group-guid')`)
				Expect(err).NotTo(HaveOccurred())
				err = realDb.QueryRow(`SELECT id FROM "groups" WHERE guid='src-group-guid'`).Scan(&srcGroupId)
				Expect(err).NotTo(HaveOccurred())

				By("inserting a dst group")
				_, err = realDb.Exec(`insert into "groups" (guid) values ('dst-group-guid')`)
				Expect(err).NotTo(HaveOccurred())
				err = realDb.QueryRow(`SELECT id FROM "groups" WHERE guid='dst-group-guid'`).Scan(&dstGroupId)
				Expect(err).NotTo(HaveOccurred())

				By("inserting a destination")
				_, err = realDb.Exec(fmt.Sprintf("insert into destinations (group_id, protocol, start_port, end_port) values (%d, 'tcp', 9889, 9999)", dstGroupId))
				Expect(err).NotTo(HaveOccurred())
				err = realDb.QueryRow(`SELECT id FROM destinations WHERE start_port=9889`).Scan(&dstId)
				Expect(err).NotTo(HaveOccurred())

				By("inserting a policy")
				_, err = realDb.Exec(fmt.Sprintf("insert into policies (group_id, destination_id) values (%d, %d)", srcGroupId, dstId))
				Expect(err).NotTo(HaveOccurred())
				err = realDb.QueryRow(fmt.Sprintf(`SELECT id FROM policies WHERE destination_id=%d`, dstId)).Scan(&policyId)
				Expect(err).NotTo(HaveOccurred())

				By("trying to change the dest_id to an id that DNE")
				_, err = realDb.Exec(fmt.Sprintf("UPDATE policies SET destination_id=999 WHERE id=%d", policyId))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Or(
					ContainSubstring("violates foreign key constraint"), // postgres error
					ContainSubstring("a foreign key constraint fails"),  // mysql error
				))

				By("trying to delete the dst")
				_, err = realDb.Exec(fmt.Sprintf("DELETE FROM destinations WHERE id=%d", dstId))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Or(
					ContainSubstring("violates foreign key constraint"), // postgres error
					ContainSubstring("a foreign key constraint fails"),  // mysql error
				))
			})
		})

		Describe("V66 - delete stored procedure", func() {
			BeforeEach(func() {
				if realDb.DriverName() != "mysql" {
					Skip("skipping mysql test")
				}
			})
			It("should delete the sole stored procedure in the database", func() {

				By("Looking for existing procedures")
				migrateTo("65")
				query := fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.ROUTINES WHERE ROUTINE_SCHEMA='%s'", dbConf.DatabaseName)
				var count int
				err := realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(1))

				By("performing migration")
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(1))

				By("Confirming stored procedure was deleted")
				err = realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(0))
			})
		})

		Describe("V67 - add Dynamic ASG related tables", func() {
			var rules string
			BeforeEach(func() {
				rules = `[{"destination": "0.0.0.0/0","ports": "53","protocol": "udp"}]`
			})

			It("should migrate", func() {
				migrateTo("67")
				By("adding the security_groups table")
				_, err := realDb.Exec(fmt.Sprintf(`
					INSERT INTO security_groups
					(guid, name, rules, staging_default, running_default, staging_spaces, running_spaces)
					VALUES ('asg-guid', 'my-group', '%s', true, true, '["space-a"]', '["space-b"]')`,
					rules))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("V68 - Remove Dynamic Egress Table - apps", func() {
			It("should migrate", func() {
				table_name := "apps"
				By("performing migration")
				migrateTo("67")

				By("Looking for existing Dynamic Egress Table")
				query := fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = '%s'", table_name)
				var count int
				err := realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(1))

				By("performing migration")
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(1))

				By("Confirming table was deleted")
				err = realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(0))

			})
		})

		Describe("V69 - Remove Dynamic Egress Table - destination_metadatas", func() {
			It("should migrate", func() {
				table_name := "destination_metadatas"
				By("performing migration")
				migrateTo("68")

				By("Looking for existing Dynamic Egress Table")
				query := fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = '%s'", table_name)
				var count int
				err := realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(1))

				By("performing migration")
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(1))

				By("Confirming table was deleted")
				err = realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(0))

			})
		})

		Describe("V70 - Remove Dynamic Egress Table - ip_ranges", func() {
			It("should migrate", func() {
				table_name := "ip_ranges"
				By("performing migration")
				migrateTo("69")

				By("Looking for existing Dynamic Egress Table")
				query := fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.TABLES WHERE  TABLE_NAME = '%s'", table_name)
				var count int
				err := realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(1))

				By("performing migration")
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(1))

				By("Confirming table was deleted")
				err = realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(0))

			})
		})

		Describe("V71 - Remove Dynamic Egress Table - spaces", func() {
			It("should migrate", func() {
				table_name := "spaces"
				By("performing migration")
				migrateTo("70")

				By("Looking for existing Dynamic Egress Table")
				query := fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.TABLES WHERE  TABLE_NAME = '%s'", table_name)
				var count int
				err := realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(1))

				By("performing migration")
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(1))

				By("Confirming table was deleted")
				err = realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(0))

			})
		})

		Describe("V72 - Remove Dynamic Egress Table - egress_policies", func() {
			It("should migrate", func() {
				table_name := "egress_policies"
				By("performing migration")
				migrateTo("71")

				By("Looking for existing Dynamic Egress Table")
				query := fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.TABLES WHERE  TABLE_NAME = '%s'", table_name)
				var count int
				err := realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(1))

				By("performing migration")
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(1))

				By("Confirming table was deleted")
				err = realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(0))

			})
		})

		Describe("V73 - Remove Dynamic Egress Table - defaults", func() {
			It("should migrate", func() {
				table_name := "defaults"
				By("performing migration")
				migrateTo("72")

				By("Looking for existing Dynamic Egress Tables")
				query := fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.TABLES WHERE  TABLE_NAME = '%s'", table_name)
				var count int
				err := realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(1))

				By("performing migration")
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(1))

				By("Confirming table was deleted")
				err = realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(0))

			})
		})

		Describe("V74 - Remove Dynamic Egress Table - terminals", func() {
			It("should migrate", func() {
				table_name := "terminals"
				By("performing migration")
				migrateTo("73")

				By("Looking for existing Dynamic Egress Table")
				query := fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.TABLES WHERE  TABLE_NAME = '%s'", table_name)
				var count int
				err := realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(1))

				By("performing migration")
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(1))

				By("Confirming table was deleted")
				err = realDb.QueryRow(query).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(0))

			})
		})

		Describe("V75 - Migrate the security_groups id from int to bigint - security_groups", func() {
			It("should migrate", func() {
				tableName := "security_groups"

				By("performing migration")
				migrateTo("75")

				By("Confirming table id was modified")
				var dataType string
				queryTableIdType := fmt.Sprintf("SELECT DATA_TYPE AS dataType FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = '%s' AND COLUMN_NAME = 'id'", tableName)
				err := realDb.QueryRow(queryTableIdType).Scan(&dataType)
				Expect(err).NotTo(HaveOccurred())
				Expect(dataType).To(Equal("bigint"))
			})
		})

		Describe("V76 - Adjust the max size of the bigint data type - security_groups", func() {
			BeforeEach(func() {
				if realDb.DriverName() != "postgres" {
					Skip("skipping postgres test")
				}
			})

			It("should migrate", func() {
				tableName := "security_groups"

				By("performing migration")
				migrateTo("76")

				By("Confirming table id was modified")
				var maxvalue int
				queryTableIdType := fmt.Sprintf("SELECT max_value AS maxValue FROM PG_SEQUENCES WHERE SEQUENCENAME = '%s_id_seq'", tableName)
				err := realDb.QueryRow(queryTableIdType).Scan(&maxvalue)
				Expect(err).NotTo(HaveOccurred())
				Expect(maxvalue).To(Equal(9223372036854775807))
			})
		})

		Describe("v82-83 - multi-value indices for security_groups space bindings", func() {
			var spacesJson string
			BeforeEach(func() {
				if isPostgresOrMySQL57(realDb) {
					Skip("skipping-unsupported-indices on non-mysql or mysql-5.7 database")
				}
				migrateTo("81")
				var guids []string
				for range 149 {
					guid, err := uuid.NewV4()
					Expect(err).ToNot(HaveOccurred())
					guids = append(guids, guid.String())
				}
				array, err := json.Marshal(guids)
				spacesJson = string(array)
				Expect(err).ToNot(HaveOccurred())
			})
			It("no longer adds indices", func() {
				By("performing migration")
				migrateTo("83")

				var count int
				By("verifying no indices exist for running spaces")
				indexCountQuery := "SELECT COUNT(*) FROM information_schema.statistics WHERE table_name = 'security_groups' AND index_name = 'running_spaces_idx' AND table_schema = DATABASE()"
				err := realDb.QueryRow(indexCountQuery).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(0))

				By("verifying no indices exist for staging spaces")
				indexCountQuery = "SELECT COUNT(*) FROM information_schema.statistics WHERE table_name = 'security_groups' AND index_name = 'staging_spaces_idx' AND table_schema = DATABASE()"
				err = realDb.QueryRow(indexCountQuery).Scan(&count)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(0))
			})

			Context("when migrating with security_groups bound to >148 running spaces", func() {
				BeforeEach(func() {
					insertSql := "INSERT INTO security_groups (guid, name, running_spaces) VALUES(?, ?, ?)"
					_, err := realDb.Exec(insertSql, "fake-guid", "my-asg", spacesJson)
					Expect(err).NotTo(HaveOccurred())
				})
				It("doesn't fail", func() {
					migrateTo("83")
				})
			})

			Context("when migrating with security_groups bound to >148 staging spaces", func() {
				BeforeEach(func() {
					insertSql := "INSERT INTO security_groups (guid, name, staging_spaces) VALUES(?, ?, ?)"
					_, err := realDb.Exec(insertSql, "fake-guid", "my-asg", spacesJson)
					Expect(err).NotTo(HaveOccurred())
				})
				It("doesn't fail", func() {
					migrateTo("83")
				})
			})
		})

		Describe("v86-91 - removing multi-value indices for security_groups space bindings", func() {

			BeforeEach(func() {
				if isPostgresOrMySQL57(realDb) {
					Skip("skipping-unsupported-indices on non-mysql or mysql-5.7 database")
				}
				migrateTo("83")
			})
			Context("when v82-83 had created multi-value indices already", func() {
				BeforeEach(func() {
					runningIndex := `CREATE INDEX running_spaces_idx ON security_groups ((CAST(running_spaces -> '$[*]' AS CHAR(36) ARRAY)))`
					_, err := realDb.Exec(runningIndex)
					Expect(err).NotTo(HaveOccurred())

					stagingIndex := `CREATE INDEX staging_spaces_idx ON security_groups ((CAST(staging_spaces -> '$[*]' AS CHAR(36) ARRAY)))`
					_, err = realDb.Exec(stagingIndex)
					Expect(err).NotTo(HaveOccurred())

					var count int
					By("verifying the index exists for running spaces")
					indexCountQuery := "SELECT COUNT(*) FROM information_schema.statistics WHERE table_name = 'security_groups' AND index_name = 'running_spaces_idx' AND table_schema = DATABASE()"
					err = realDb.QueryRow(indexCountQuery).Scan(&count)
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(1))

					By("verifying the index exists for staging spaces")
					indexCountQuery = "SELECT COUNT(*) FROM information_schema.statistics WHERE table_name = 'security_groups' AND index_name = 'staging_spaces_idx' AND table_schema = DATABASE()"
					err = realDb.QueryRow(indexCountQuery).Scan(&count)
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(1))
				})

				It("removes the multi-value indices from previous versions of v82+83", func() {
					By("performing migration")
					migrateTo("91")

					var count int
					By("verifying no indices exist for running spaces")
					indexCountQuery := "SELECT COUNT(*) FROM information_schema.statistics WHERE table_name = 'security_groups' AND index_name = 'running_spaces_idx' AND table_schema = DATABASE()"
					err := realDb.QueryRow(indexCountQuery).Scan(&count)
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(0))

					By("verifying no indices exist for staging spaces")
					indexCountQuery = "SELECT COUNT(*) FROM information_schema.statistics WHERE table_name = 'security_groups' AND index_name = 'staging_spaces_idx' AND table_schema = DATABASE()"
					err = realDb.QueryRow(indexCountQuery).Scan(&count)
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(0))
				})
			})

			Context("if no previous indices were present", func() {
				It("doesn't fail", func() {
					var count int
					By("verifying no indices exist for running spaces")
					indexCountQuery := "SELECT COUNT(*) FROM information_schema.statistics WHERE table_name = 'security_groups' AND index_name = 'running_spaces_idx' AND table_schema = DATABASE()"
					err := realDb.QueryRow(indexCountQuery).Scan(&count)
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(0))

					By("verifying no indices exist for staging spaces")
					indexCountQuery = "SELECT COUNT(*) FROM information_schema.statistics WHERE table_name = 'security_groups' AND index_name = 'staging_spaces_idx' AND table_schema = DATABASE()"
					err = realDb.QueryRow(indexCountQuery).Scan(&count)
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(0))

					By("migrating after ensuring the indices were absent")
					migrateTo("91")
				})
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

		Describe("v92-93 - adding new association tables", func() {

			It("succeeds", func() {
				migrateTo("93")
			})
		})
		Describe("v94", func() {
			It("adds a `hash` column to security_groups", func() {
				migrateTo("94")
				_, err := realDb.Query("SELECT hash from security_groups")
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Describe("v95-100 - migrating json running_spaces/staging_spaces to join tables", func() {
			BeforeEach(func() {
				migrateTo("94")

				var rows int
				err := realDb.QueryRow("SELECT COUNT(*) FROM security_groups").Scan(&rows)
				Expect(err).NotTo(HaveOccurred())
				Expect(rows).To(Equal(0))
				err = realDb.QueryRow("SELECT COUNT(*) FROM running_security_groups_spaces").Scan(&rows)
				Expect(err).NotTo(HaveOccurred())
				Expect(rows).To(Equal(0))
				err = realDb.QueryRow("SELECT COUNT(*) FROM staging_security_groups_spaces").Scan(&rows)
				Expect(err).NotTo(HaveOccurred())
				Expect(rows).To(Equal(0))
			})
			AfterEach(func() {
				_, err := realDb.Exec("DELETE FROM security_groups")
				Expect(err).NotTo(HaveOccurred())

				_, err = realDb.Exec("DELETE FROM running_security_groups_spaces")
				Expect(err).NotTo(HaveOccurred())
				_, err = realDb.Exec("DELETE FROM staging_security_groups_spaces")
				Expect(err).NotTo(HaveOccurred())
			})
			Context("when no rows exist in the security_groups table", func() {
				It("migrates without error", func() {
					migrateTo("100")
				})
			})
			Context("when a security group has running spaces but no staging spaces", func() {
				BeforeEach(func() {
					_, err := realDb.Exec(
						`INSERT INTO security_groups (name, guid, running_spaces, staging_spaces) VALUES
							('guid-1', 'guid-1', JSON_ARRAY('space-1', 'space-2'), JSON_ARRAY())`)
					Expect(err).NotTo(HaveOccurred())
				})
				It("adds entries to the running_space join table, but nothing to the staging space join table", func() {
					migrateTo("100")

					ExpectAssociatedSpacesToConsistOf(realDb, "running", []JoinRow{{
						SecurityGroup: "guid-1",
						Space:         "space-1",
					}, {
						SecurityGroup: "guid-1",
						Space:         "space-2",
					}})
					ExpectAssociatedSpacesToConsistOf(realDb, "staging", []JoinRow{})
				})
			})
			Context("when a security group has staging spaces but no running spaces", func() {
				BeforeEach(func() {
					_, err := realDb.Exec(
						`INSERT INTO security_groups (name, guid, running_spaces, staging_spaces) VALUES
							('guid-1', 'guid-1', JSON_ARRAY(), JSON_ARRAY('space-1', 'space-2'))`)
					Expect(err).NotTo(HaveOccurred())
				})
				It("adds entries to the staging_space join table, but nothing to the running space join table", func() {
					migrateTo("100")

					ExpectAssociatedSpacesToConsistOf(realDb, "staging", []JoinRow{{
						SecurityGroup: "guid-1",
						Space:         "space-1",
					}, {
						SecurityGroup: "guid-1",
						Space:         "space-2",
					}})
					ExpectAssociatedSpacesToConsistOf(realDb, "running", []JoinRow{})
				})
			})
			Context("when a security group has no staging spaces or no running spaces", func() {
				BeforeEach(func() {
					_, err := realDb.Exec(
						`INSERT INTO security_groups (name, guid, running_spaces, staging_spaces) VALUES
							('guid-1', 'guid-1', JSON_ARRAY(), JSON_ARRAY())`)
					Expect(err).NotTo(HaveOccurred())
				})
				It("adds nothing to the join tables", func() {
					migrateTo("100")

					ExpectAssociatedSpacesToConsistOf(realDb, "running", []JoinRow{})
					ExpectAssociatedSpacesToConsistOf(realDb, "staging", []JoinRow{})
				})
			})
			Context("when a security group has staging spaces and running spaces", func() {
				BeforeEach(func() {
					_, err := realDb.Exec(
						`INSERT INTO security_groups (name, guid, running_spaces, staging_spaces) VALUES
							('guid-1', 'guid-1', JSON_ARRAY('space-1', 'space-2', 'common-space-1'), JSON_ARRAY('space-3', 'space-4', 'common-space-1'))`)
					Expect(err).NotTo(HaveOccurred())
				})
				It("adds the security group to both tables with the correct running + staging spaces in each", func() {
					migrateTo("100")

					ExpectAssociatedSpacesToConsistOf(realDb, "staging", []JoinRow{{
						SecurityGroup: "guid-1",
						Space:         "space-3",
					}, {
						SecurityGroup: "guid-1",
						Space:         "space-4",
					}, {
						SecurityGroup: "guid-1",
						Space:         "common-space-1",
					}})
					ExpectAssociatedSpacesToConsistOf(realDb, "running", []JoinRow{{
						SecurityGroup: "guid-1",
						Space:         "space-1",
					}, {
						SecurityGroup: "guid-1",
						Space:         "space-2",
					}, {
						SecurityGroup: "guid-1",
						Space:         "common-space-1",
					}})
				})
			})
			Context("when the longest number of spaces associated with a security group is in the running_spaces column", func() {
				BeforeEach(func() {
					_, err := realDb.Exec(
						`INSERT INTO security_groups (name, guid, running_spaces, staging_spaces) VALUES
							('guid-1', 'guid-1', JSON_ARRAY('space-1', 'space-2', 'space-3'), JSON_ARRAY('space-3', 'space-4')),
							('guid-2', 'guid-2', JSON_ARRAY('space-1', 'space-2'), JSON_ARRAY('space-3', 'space-4'))
						`)
					Expect(err).NotTo(HaveOccurred())
				})
				It("is able to migrate successfully", func() {
					migrateTo("100")
				})
			})
			Context("when the longest number of spaces associated with a security group is in the staginging_spaces column", func() {
				BeforeEach(func() {
					_, err := realDb.Exec(
						`INSERT INTO security_groups (name, guid, running_spaces, staging_spaces) VALUES
							('guid-1', 'guid-1', JSON_ARRAY('space-1', 'space-2', 'space-3'), JSON_ARRAY('space-3', 'space-4')),
							('guid-2', 'guid-2', JSON_ARRAY('space-1', 'space-2'), JSON_ARRAY('space-3', 'space-4', 'space-5', 'space-6'))
						`)
					Expect(err).NotTo(HaveOccurred())
				})
				It("is able to migrate successfully", func() {
					migrateTo("100")
				})
			})
			Context("when multiple rows have multiple spaces bound", func() {
				BeforeEach(func() {
					_, err := realDb.Exec(
						`INSERT INTO security_groups (name, guid, running_spaces, staging_spaces) VALUES
							('guid-1', 'guid-1', JSON_ARRAY('space-1', 'space-2', 'space-3'), JSON_ARRAY('space-3', 'space-4')),
							('guid-2', 'guid-2', JSON_ARRAY('space-1', 'space-2'), JSON_ARRAY('space-3', 'space-4', 'space-5', 'space-6')),
							('guid-3', 'guid-3', JSON_ARRAY(), JSON_ARRAY()),
							('guid-4', 'guid-4', JSON_ARRAY(), JSON_ARRAY('space-10')),
							('guid-5', 'guid-5', JSON_ARRAY('space-11'), JSON_ARRAY())
						`)
					Expect(err).NotTo(HaveOccurred())
				})
				It("adds entries into all relevant join records", func() {
					migrateTo("100")

					ExpectAssociatedSpacesToConsistOf(realDb, "staging", []JoinRow{{
						SecurityGroup: "guid-1",
						Space:         "space-3",
					}, {
						SecurityGroup: "guid-1",
						Space:         "space-4",
					}, {
						SecurityGroup: "guid-2",
						Space:         "space-3",
					}, {
						SecurityGroup: "guid-2",
						Space:         "space-4",
					}, {
						SecurityGroup: "guid-2",
						Space:         "space-5",
					}, {
						SecurityGroup: "guid-2",
						Space:         "space-6",
					}, {
						SecurityGroup: "guid-4",
						Space:         "space-10",
					}})
					ExpectAssociatedSpacesToConsistOf(realDb, "running", []JoinRow{{
						SecurityGroup: "guid-1",
						Space:         "space-1",
					}, {
						SecurityGroup: "guid-1",
						Space:         "space-2",
					}, {
						SecurityGroup: "guid-1",
						Space:         "space-3",
					}, {
						SecurityGroup: "guid-2",
						Space:         "space-1",
					}, {
						SecurityGroup: "guid-2",
						Space:         "space-2",
					}, {
						SecurityGroup: "guid-5",
						Space:         "space-11",
					}})
				})
			})
		})
	})

	Describe("Down Migration", func() {
		It("should no-op", func() {
			adapter := migrations.MigrateAdapter{}

			toMigrate := migrations.V1ModifiedMigrationsToPerform
			migrateUp(realDb, adapter, toMigrate)
			toMigrate = append(toMigrate, migrations.V2ModifiedMigrationsToPerform...)
			migrateUp(realDb, adapter, toMigrate)
			toMigrate = append(toMigrate, migrations.V3ModifiedMigrationsToPerform...)
			migrateUp(realDb, adapter, toMigrate)
			toMigrate = append(toMigrate, migrations.MigrationsToPerform...)
			migrateUp(realDb, adapter, toMigrate)

			migrateDown(realDb, adapter, migrations.V1ModifiedMigrationsToPerform)
			migrateDown(realDb, adapter, migrations.V2ModifiedMigrationsToPerform)
			migrateDown(realDb, adapter, migrations.V3ModifiedMigrationsToPerform)
			migrateDown(realDb, adapter, migrations.MigrationsToPerform)
		})
	})

	Describe("Migrations should be atomic", func() {
		It("should contain a single statement per migration", func() {
			for _, migration := range migrations.MigrationsToPerform {
				for dbType, statements := range migration.Up {
					if len(statements) > 1 {
						Fail(fmt.Sprintf("Migration %s for %s has %d statements. Expected a single statement per migration.",
							migration.Id, dbType, len(statements)))
					}
				}
			}
		})
	})
})

func migrateUp(realDb *db.ConnWrapper, adapter migrations.MigrateAdapter, migrationGroup migrations.PolicyServerMigrations) {
	_, err := adapter.ExecMax(
		realDb,
		realDb.DriverName(),
		migrate.MemoryMigrationSource{
			Migrations: migrationGroup.ForDriver(realDb.DriverName()),
		},
		migrate.Up,
		0,
	)
	Expect(err).NotTo(HaveOccurred())
}

func migrateDown(realDb *db.ConnWrapper, adapter migrations.MigrateAdapter, migrationGroup migrations.PolicyServerMigrations) {
	numberOfMigrations, err := adapter.ExecMax(
		realDb,
		realDb.DriverName(),
		migrate.MemoryMigrationSource{
			Migrations: migrationGroup.ForDriver(realDb.DriverName()),
		},
		migrate.Down,
		0,
	)
	Expect(err).To(MatchError("down migration not supported"))
	Expect(numberOfMigrations).To(Equal(0))
}

func expectMigrations(realDb *db.ConnWrapper, expectedMigrations []string) {
	rows, err := realDb.Query(`select ID from gorp_migrations`)
	Expect(err).NotTo(HaveOccurred())
	defer rows.Close()
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

func insertTerminal(realDb *db.ConnWrapper) int64 {
	var terminalId int64
	if realDb.DriverName() == "mysql" {
		result, err := realDb.Exec("INSERT INTO terminals (id) VALUES (NULL)")
		Expect(err).NotTo(HaveOccurred())
		terminalId, err = result.LastInsertId()
		Expect(err).NotTo(HaveOccurred())
	} else {
		err := realDb.QueryRow("INSERT INTO terminals DEFAULT VALUES RETURNING id").Scan(&terminalId)
		Expect(err).NotTo(HaveOccurred())
	}
	return terminalId
}

func queryTableForColumnValues(tableName, columnName string, realDb *db.ConnWrapper) []string {
	rows, err := realDb.Query(helpers.RebindForSQLDialect(fmt.Sprintf(`
		select %s from %s
	`, columnName, tableName), realDb.DriverName()))
	Expect(err).NotTo(HaveOccurred())

	defer rows.Close()
	var values []string
	for rows.Next() {
		var value string
		Expect(rows.Scan(&value)).To(Succeed())
		values = append(values, value)
	}
	return values
}

func queryTableColumnNames(tableName string, realDb *db.ConnWrapper) []string {
	rows, err := realDb.Query(realDb.Rebind(helpers.RebindForSQLDialect(`
		select COLUMN_NAME
		from INFORMATION_SCHEMA.COLUMNS t1
		where TABLE_NAME = ?
	`, realDb.DriverName())), tableName)
	Expect(err).NotTo(HaveOccurred())

	columns := []string{}
	defer rows.Close()
	for rows.Next() {
		var columnName string
		Expect(rows.Scan(&columnName)).To(Succeed())
		columns = append(columns, columnName)
	}

	return columns
}

func getMigrationIndex(migrationsProvider *migrations.MigrationsProvider, migrationId string) int {
	migrationsToPerform, err := migrationsProvider.MigrationsToPerform()
	Expect(err).NotTo(HaveOccurred())
	for i, migration := range migrationsToPerform {
		if migration.Id == migrationId {
			return i + 1
		}
	}
	Fail("couldn't find migration with id: " + migrationId)
	return -1
}

func isPostgresOrMySQL57(realDb *db.ConnWrapper) bool {
	if realDb.DriverName() == "mysql" {
		row := realDb.DB.QueryRow("SELECT VERSION()")
		// if no rows are returned, we're definitely not mysql 8, so short circuit out
		if row != nil {
			var version string
			err := row.Scan(&version)
			Expect(err).NotTo(HaveOccurred())

			// mysql returns major.minor.patch version number strings
			// e.g. `5.7.43` would be the data returned for the VERSION() query
			if strings.HasPrefix(version, "5.7.") {
				return true
			} else {
				return false
			}
		}
	}
	return true
}

type JoinRow struct {
	SecurityGroup string
	Space         string
}

func ExpectAssociatedSpacesToConsistOf(realDb *db.ConnWrapper, table string, expectedRows []JoinRow) {
	rows, err := realDb.Query(fmt.Sprintf("SELECT security_group_guid, space_guid FROM %s_security_groups_spaces", table))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	if len(expectedRows) == 0 {
		ExpectWithOffset(1, scanCountRow(rows)).To(Equal(0))
	} else {
		var receivedRows []JoinRow
		for rows.Next() {
			var sg, space string
			err := rows.Scan(&sg, &space)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			receivedRows = append(receivedRows, JoinRow{
				SecurityGroup: sg,
				Space:         space,
			})
		}
		ExpectWithOffset(1, receivedRows).To(ConsistOf(expectedRows))
	}
}
