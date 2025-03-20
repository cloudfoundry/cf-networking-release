package migrations

import (
	"database/sql"
	"fmt"
	"strings"

	migrate "github.com/cf-container-networking/sql-migrate"
	"github.com/jmoiron/sqlx"
)

//go:generate counterfeiter -generate

//counterfeiter:generate -o fakes/migrate_adapter.go --fake-name MigrateAdapter . migrateAdapter
type migrateAdapter interface {
	ExecMax(db MigrationDb, dialect string, m migrate.MigrationSource, dir migrate.MigrationDirection, maxNumMigrations int) (int, error)
}

//counterfeiter:generate -o fakes/migration_db.go --fake-name MigrationDb . MigrationDb
type MigrationDb interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	DriverName() string
	RawConnection() *sqlx.DB
}

//counterfeiter:generate -o fakes/migrations_provider.go --fake-name MigrationsProvider . migrationsProvider
type migrationsProvider interface {
	MigrationsToPerform(bool) (PolicyServerMigrations, error)
}

type Migrator struct {
	MigrateAdapter     migrateAdapter
	MigrationsProvider migrationsProvider
}

func (m *Migrator) PerformMigrations(driverName string, migrationDb MigrationDb, maxNumMigrations int) (int, error) {
	var mysql57 bool
	if driverName == "mysql" {
		row := migrationDb.QueryRow("SELECT VERSION()")
		// if no rows are returned, we're definitely not mysql 5.7, so short circuit out
		if row != nil {
			var version string
			err := row.Scan(&version)
			if err != nil {
				return 0, fmt.Errorf("Unable to detect MySQL Version: %s", err)
			}
			// mysql returns major.minor.patch version number strings
			// e.g. `5.7.43` would be the data returned for the VERSION() query
			mysql57 = strings.HasPrefix(version, "5.7.")
		}
	}
	// at this point, the mysql57 bool is false if postgres, false if no rows were returned
	// for SELECT VERSION(), false if the prefix doesn't start with `5.7.` (in case there's ever a 5.70)
	// and true if 5.7.x. the above should only have thrown an error if there was a problem talking
	// to the database to execute the query, or the row couldn't be marshalled into a string variable

	migrationsToPerform, err := m.MigrationsProvider.MigrationsToPerform(mysql57)
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
	Id          string
	SkipMySQL57 bool
	Up          map[string][]string
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
