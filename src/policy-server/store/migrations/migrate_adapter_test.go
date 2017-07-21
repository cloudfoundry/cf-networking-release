package migrations_test

import (
	"policy-server/store/migrations"
	"policy-server/store/migrations/fakes"

	"github.com/cf-container-networking/sql-migrate"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MigrateAdapter", func() {

	var (
		migrateAdapter *migrations.MigrateAdapter
	)

	BeforeEach(func() {
		migrateAdapter = &migrations.MigrateAdapter{}
	})

	Describe("ExecMax", func() {
		Context("when the passed in database is not a sqlx.DB", func() {
			It("returns an error", func() {
				fakeMigrationDb := &fakes.MigrationDb{}
				_, err := migrateAdapter.ExecMax(fakeMigrationDb, "some-dialect", migrate.MemoryMigrationSource{}, migrate.Up, 0)
				Expect(err).To(MatchError("unable to adapt for db migration"))
			})
		})
	})
})
