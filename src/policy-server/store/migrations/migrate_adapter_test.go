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
		Context("when the migration direction is down", func() {
			It("returns an error", func() {
				fakeMigrationDb := &fakes.MigrationDb{}
				_, err := migrateAdapter.ExecMax(fakeMigrationDb, "some-dialect", migrate.MemoryMigrationSource{}, migrate.Down, 0)
				Expect(err).To(MatchError("down migration not supported"))
			})
		})
	})
})
