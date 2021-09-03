package migrations

import "fmt"

//go:generate counterfeiter -o fakes/migration_store.go --fake-name MigrationStore . migrationStore
type migrationStore interface {
	HasV1MigrationOccurred() (bool, error)
	HasV2MigrationOccurred() (bool, error)
	HasV3MigrationOccurred() (bool, error)
}

type MigrationsProvider struct {
	Store migrationStore
}

func (m *MigrationsProvider) MigrationsToPerform() (PolicyServerMigrations, error) {
	policyServerMigrations := PolicyServerMigrations{}

	hasV1, err := m.Store.HasV1MigrationOccurred()
	if err != nil {
		return PolicyServerMigrations{}, fmt.Errorf("failed to check V1 Migration status: %s", err)
	}

	if hasV1 {
		policyServerMigrations = append(
			policyServerMigrations,
			V1LegacyMigrationsToPerform...,
		)
	} else {
		policyServerMigrations = append(
			policyServerMigrations,
			V1ModifiedMigrationsToPerform...,
		)
	}

	hasV2, err := m.Store.HasV2MigrationOccurred()
	if err != nil {
		return PolicyServerMigrations{}, fmt.Errorf("failed to check V2 Migration status: %s", err)
	}

	if hasV2 {
		policyServerMigrations = append(
			policyServerMigrations,
			V2LegacyMigrationsToPerform...,
		)
	} else {
		policyServerMigrations = append(
			policyServerMigrations,
			V2ModifiedMigrationsToPerform...,
		)
	}

	hasV3, err := m.Store.HasV3MigrationOccurred()
	if err != nil {
		return PolicyServerMigrations{}, fmt.Errorf("failed to check V3 Migration status: %s", err)
	}

	if hasV3 {
		policyServerMigrations = append(
			policyServerMigrations,
			V3LegacyMigrationsToPerform...,
		)
	} else {
		policyServerMigrations = append(
			policyServerMigrations,
			V3ModifiedMigrationsToPerform...,
		)
	}

	policyServerMigrations = append(
		policyServerMigrations,
		MigrationsToPerform...,
	)

	return policyServerMigrations, nil
}
