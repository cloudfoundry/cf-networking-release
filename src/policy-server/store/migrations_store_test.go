package store_test

import (
	"fmt"
	"policy-server/db"
	"policy-server/store"
	"policy-server/store/migrations"
	"test-helpers"
	"time"

	migrationsFakes "policy-server/store/migrations/fakes"

	dbHelper "code.cloudfoundry.org/cf-networking-helpers/db"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport"
	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("MigrationsStore", func() {
	var (
		dbConf dbHelper.Config
		realDb *db.ConnWrapper

		migrationsProvider *migrationsFakes.MigrationsProvider
		migrator           *migrations.Migrator

		migrationsStore *store.MigrationsStore
	)

	BeforeEach(func() {
		dbConf = testsupport.GetDBConfig()
		dbConf.DatabaseName = fmt.Sprintf("migrator_store_test_node_%d", time.Now().UnixNano())
		dbConf.Timeout = 30
		testhelpers.CreateDatabase(dbConf)

		logger := lager.NewLogger("Migrations Store Test")

		realDb = db.NewConnectionPool(dbConf, 200, 200, "Migrations Store Test", "Migrations Store Test", logger)

		migrationsProvider = &migrationsFakes.MigrationsProvider{}

		migrator = &migrations.Migrator{
			MigrateAdapter:     &migrations.MigrateAdapter{},
			MigrationsProvider: migrationsProvider,
		}

		migrationsStore = &store.MigrationsStore{
			DBConn: realDb,
		}
	})

	AfterEach(func() {
		if realDb != nil {
			Expect(realDb.Close()).To(Succeed())
		}
		testhelpers.RemoveDatabase(dbConf)
	})

	Describe("HasV1MigrationOccurred", func() {
		Context("when modified v1 and v1a, but not v1b have run", func() {
			BeforeEach(func() {
				m := migrations.V1ModifiedMigrationsToPerform[0:2]
				migrationsProvider.MigrationsToPerformReturns(m, nil)

				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(2))
			})

			It("returns false", func() {
				hasOccurred, err := migrationsStore.HasV1MigrationOccurred()
				Expect(err).NotTo(HaveOccurred())
				Expect(hasOccurred).To(BeFalse())
			})
		})

		Context("when v1 migrations have occurred", func() {
			BeforeEach(func() {
				m := append(migrations.V1LegacyMigrationsToPerform,
					migrations.MigrationsToPerform...)
				migrationsProvider.MigrationsToPerformReturns(m, nil)

				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(1))
			})

			It("returns true", func() {
				hasOccurred, err := migrationsStore.HasV1MigrationOccurred()
				Expect(err).NotTo(HaveOccurred())
				Expect(hasOccurred).To(BeTrue())
			})
		})

		It("returns false if v1 migration has not occurred", func() {
			hasOccurred, err := migrationsStore.HasV1MigrationOccurred()
			Expect(err).NotTo(HaveOccurred())
			Expect(hasOccurred).To(BeFalse())
		})
	})

	Describe("HasV2MigrationOccurred", func() {
		BeforeEach(func() {
			m := append(migrations.V1LegacyMigrationsToPerform,
				migrations.V2LegacyMigrationsToPerform...)
			m = append(m,
				migrations.V3LegacyMigrationsToPerform...)
			m = append(m,
				migrations.MigrationsToPerform...)
			migrationsProvider.MigrationsToPerformReturns(m, nil)
		})

		Context("when v2 migrations has occurred", func() {
			BeforeEach(func() {
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(10))
			})

			It("returns true", func() {
				hasOccurred, err := migrationsStore.HasV2MigrationOccurred()
				Expect(err).NotTo(HaveOccurred())
				Expect(hasOccurred).To(BeTrue())
			})
		})

		Context("when v2 migration has not occurred", func() {
			BeforeEach(func() {
				numV1Migrations := len(migrations.V1LegacyMigrationsToPerform)
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, numV1Migrations)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(numMigrations))
			})

			It("returns false", func() {
				hasOccurred, err := migrationsStore.HasV2MigrationOccurred()
				Expect(err).NotTo(HaveOccurred())
				Expect(hasOccurred).To(BeFalse())
			})
		})

		Context("when modified v2 through v2e, but not v2f have run", func() {
			BeforeEach(func() {
				m := append(migrations.V1ModifiedMigrationsToPerform,
					migrations.V2ModifiedMigrationsToPerform...)
				m = m[0 : len(m)-1]
				migrationsProvider.MigrationsToPerformReturns(m, nil)

				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(9))
			})

			It("returns false", func() {
				hasOccurred, err := migrationsStore.HasV2MigrationOccurred()
				Expect(err).NotTo(HaveOccurred())
				Expect(hasOccurred).To(BeFalse())
			})
		})
	})

	Describe("HasV3MigrationOccurred", func() {
		BeforeEach(func() {
			m := append(migrations.V1LegacyMigrationsToPerform,
				migrations.V2LegacyMigrationsToPerform...)
			m = append(m,
				migrations.V3LegacyMigrationsToPerform...)
			m = append(m,
				migrations.MigrationsToPerform...)
			migrationsProvider.MigrationsToPerformReturns(m, nil)
		})

		Context("when v3 migrations has occurred", func() {
			BeforeEach(func() {
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 12)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(12))
			})

			It("returns true", func() {
				hasOccurred, err := migrationsStore.HasV3MigrationOccurred()
				Expect(err).NotTo(HaveOccurred())
				Expect(hasOccurred).To(BeTrue())
			})
		})

		Context("when v3 migration has not occurred", func() {
			BeforeEach(func() {
				numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(numMigrations).To(Equal(10))
			})

			It("returns false", func() {
				hasOccurred, err := migrationsStore.HasV3MigrationOccurred()
				Expect(err).NotTo(HaveOccurred())
				Expect(hasOccurred).To(BeFalse())
			})
		})
	})

	DescribeTable("Partial Modified Migrations", func(migrationsToRun int, hasOccurredFunc func() (bool, error)) {
		m := append(migrations.V1ModifiedMigrationsToPerform,
			migrations.V2ModifiedMigrationsToPerform...)
		m = append(m,
			migrations.V3ModifiedMigrationsToPerform...)
		m = append(m,
			migrations.MigrationsToPerform...)

		migrationsProvider.MigrationsToPerformReturns(m, nil)

		numMigrations, err := migrator.PerformMigrations(realDb.DriverName(), realDb, migrationsToRun)
		Expect(err).NotTo(HaveOccurred())
		Expect(numMigrations).To(Equal(migrationsToRun))

		hasOccurred, err := hasOccurredFunc()
		Expect(err).NotTo(HaveOccurred())
		Expect(hasOccurred).To(BeFalse())
	},
		Entry("1 of 3 v1 modified migrations succeeded", 1, func() (bool, error) { return migrationsStore.HasV1MigrationOccurred() }),
		Entry("2 of 3 v1 modified migrations succeeded", 2, func() (bool, error) { return migrationsStore.HasV1MigrationOccurred() }),
		Entry("1 of 7 v2 modified migrations succeeded", 4, func() (bool, error) { return migrationsStore.HasV2MigrationOccurred() }),
		Entry("1 of 2 v3 modified migrations succeeded", 11, func() (bool, error) { return migrationsStore.HasV3MigrationOccurred() }),
	)
})
