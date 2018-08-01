package migrations

import (
	"database/sql"
	"fmt"

	"github.com/cf-container-networking/sql-migrate"
	"github.com/jmoiron/sqlx"
)

//go:generate counterfeiter -o fakes/migrate_adapter.go --fake-name MigrateAdapter . migrateAdapter
type migrateAdapter interface {
	ExecMax(db MigrationDb, dialect string, m migrate.MigrationSource, dir migrate.MigrationDirection, maxNumMigrations int) (int, error)
}

//go:generate counterfeiter -o fakes/migration_db.go --fake-name MigrationDb . MigrationDb
type MigrationDb interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	DriverName() string
	RawConnection() *sqlx.DB
}

//go:generate counterfeiter -o fakes/migrations_provider.go --fake-name MigrationsProvider . migrationsProvider
type migrationsProvider interface {
	MigrationsToPerform() (PolicyServerMigrations, error)
}

type Migrator struct {
	MigrateAdapter     migrateAdapter
	MigrationsProvider migrationsProvider
}

func (m *Migrator) PerformMigrations(driverName string, migrationDb MigrationDb, maxNumMigrations int) (int, error) {
	migrationsToPerform, err := m.MigrationsProvider.MigrationsToPerform()
	if err != nil {
		return 0, fmt.Errorf("error retrieving migrations to perform: %s", err)
	}

	if !migrationsToPerform.supportsDriver(driverName) {
		return 0, fmt.Errorf("unsupported driver: %s", driverName)
	}

	numMigrations, err := m.MigrateAdapter.ExecMax(
		migrationDb,
		driverName,
		migrate.MemoryMigrationSource{
			Migrations: migrationsToPerform.ForDriver(driverName),
		},
		migrate.Up,
		maxNumMigrations,
	)

	if err != nil {
		return numMigrations, fmt.Errorf("executing migration: %s", err)
	}
	return numMigrations, nil
}

type PolicyServerMigrations []PolicyServerMigration

func (s PolicyServerMigrations) ForDriver(driverName string) []*migrate.Migration {
	migrationMapped := []*migrate.Migration{}

	for _, migration := range s {
		migrationMapped = append(migrationMapped, migration.forDriver(driverName))
	}
	return migrationMapped
}

func (s PolicyServerMigrations) supportsDriver(driverName string) bool {
	for _, migration := range s {
		if !migration.supportsDriver(driverName) {
			return false
		}
	}
	return true
}

type PolicyServerMigration struct {
	Id string
	Up map[string][]string
}

func (psm *PolicyServerMigration) forDriver(driverName string) *migrate.Migration {
	return &migrate.Migration{
		Id: psm.Id,
		Up: psm.Up[driverName],
	}
}

func (psm *PolicyServerMigration) supportsDriver(driverName string) bool {
	_, foundUp := psm.Up[driverName]
	return foundUp
}
