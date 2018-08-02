package migrations_test

import (
	"errors"
	"policy-server/store/migrations"
	"policy-server/store/migrations/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gmmatchers "github.com/pivotal-cf-experimental/gomegamatchers"
)

var _ = Describe("Migrations Provider", func() {
	var (
		migrationStore     *fakes.MigrationStore
		migrationsProvider *migrations.MigrationsProvider
	)

	BeforeEach(func() {
		migrationStore = &fakes.MigrationStore{}
		migrationsProvider = &migrations.MigrationsProvider{
			Store: migrationStore,
		}
	})

	It("returns a list of migrations to perform", func() {
		migrationsToPerform, err := migrationsProvider.MigrationsToPerform()
		Expect(err).ToNot(HaveOccurred())
		expectedMigrations := migrations.PolicyServerMigrations{
			migrations.V1ModifiedMigrationsToPerform[0],
			migrations.V1ModifiedMigrationsToPerform[1],
			migrations.V1ModifiedMigrationsToPerform[2],
			migrations.V2ModifiedMigrationsToPerform[0],
			migrations.V2ModifiedMigrationsToPerform[1],
			migrations.V2ModifiedMigrationsToPerform[2],
			migrations.V2ModifiedMigrationsToPerform[3],
			migrations.V2ModifiedMigrationsToPerform[4],
			migrations.V2ModifiedMigrationsToPerform[5],
			migrations.V2ModifiedMigrationsToPerform[6],
			migrations.V3ModifiedMigrationsToPerform[0],
			migrations.V3ModifiedMigrationsToPerform[1],
		}
		expectedMigrations = append(expectedMigrations, migrations.MigrationsToPerform...)
		Expect(migrationsToPerform).To(Equal(expectedMigrations))
	})

	It("returns a helpful error message for V1MigrationOccurred errors", func() {
		migrationStore.HasV1MigrationOccurredReturns(false, errors.New("I AM ERROR"))
		_, err := migrationsProvider.MigrationsToPerform()

		Expect(err).To(MatchError("failed to check V1 Migration status: I AM ERROR"))
	})

	It("returns a helpful error message for V2MigrationOccurred errors", func() {
		migrationStore.HasV2MigrationOccurredReturns(false, errors.New("I AM ERROR"))
		_, err := migrationsProvider.MigrationsToPerform()

		Expect(err).To(MatchError("failed to check V2 Migration status: I AM ERROR"))
	})

	It("returns a helpful error message for V3MigrationOccurred errors", func() {
		migrationStore.HasV3MigrationOccurredReturns(false, errors.New("I AM ERROR"))
		_, err := migrationsProvider.MigrationsToPerform()

		Expect(err).To(MatchError("failed to check V3 Migration status: I AM ERROR"))
	})

	Context("when legacy v1 migration has already occurred", func() {
		BeforeEach(func() {
			migrationStore.HasV1MigrationOccurredReturns(true, nil)
		})

		It("returns a legacy v1 migration in the list of migrations to perform", func() {
			migrationsToPerform, err := migrationsProvider.MigrationsToPerform()
			Expect(err).ToNot(HaveOccurred())
			expectedMigrations := migrations.PolicyServerMigrations{
				migrations.V1LegacyMigrationsToPerform[0],
				migrations.V1LegacyMigrationsToPerform[1],
				migrations.V1LegacyMigrationsToPerform[2],
				migrations.V2ModifiedMigrationsToPerform[0],
				migrations.V2ModifiedMigrationsToPerform[1],
				migrations.V2ModifiedMigrationsToPerform[2],
				migrations.V2ModifiedMigrationsToPerform[3],
				migrations.V2ModifiedMigrationsToPerform[4],
				migrations.V2ModifiedMigrationsToPerform[5],
				migrations.V2ModifiedMigrationsToPerform[6],
				migrations.V3ModifiedMigrationsToPerform[0],
				migrations.V3ModifiedMigrationsToPerform[1],
			}
			expectedMigrations = append(expectedMigrations, migrations.MigrationsToPerform...)
			Expect(migrationsToPerform).To(Equal(expectedMigrations))
		})
	})

	Context("when legacy v2 migration has already occurred", func() {
		BeforeEach(func() {
			migrationStore.HasV2MigrationOccurredReturns(true, nil)
		})

		It("returns a legacy v2 migration in the list of migrations to perform", func() {
			migrationsToPerform, err := migrationsProvider.MigrationsToPerform()
			Expect(err).ToNot(HaveOccurred())
			expectedMigrations := migrations.PolicyServerMigrations{
				migrations.V2LegacyMigrationsToPerform[0],
				migrations.V2LegacyMigrationsToPerform[1],
				migrations.V2LegacyMigrationsToPerform[2],
				migrations.V2LegacyMigrationsToPerform[3],
				migrations.V2LegacyMigrationsToPerform[4],
				migrations.V2LegacyMigrationsToPerform[5],
				migrations.V2LegacyMigrationsToPerform[6],
				migrations.V3ModifiedMigrationsToPerform[0],
				migrations.V3ModifiedMigrationsToPerform[1],
			}
			expectedMigrations = append(expectedMigrations, migrations.MigrationsToPerform...)
			Expect(migrationsToPerform).To(gmmatchers.ContainSequence(expectedMigrations))
		})
	})

	Context("when legacy v3 migration has already occurred", func() {
		BeforeEach(func() {
			migrationStore.HasV3MigrationOccurredReturns(true, nil)
		})

		It("returns a legacy v3 migration in the list of migrations to perform", func() {
			migrationsToPerform, err := migrationsProvider.MigrationsToPerform()
			Expect(err).ToNot(HaveOccurred())
			expectedMigrations := migrations.PolicyServerMigrations{
				migrations.V3LegacyMigrationsToPerform[0],
				migrations.V3LegacyMigrationsToPerform[1],
			}
			expectedMigrations = append(expectedMigrations, migrations.MigrationsToPerform...)
			Expect(migrationsToPerform).To(gmmatchers.ContainSequence(expectedMigrations))
		})
	})
})
