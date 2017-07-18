package migrations

import (
	"fmt"
	"github.com/rubenv/sql-migrate"
	"database/sql"
)

//go:generate counterfeiter -o fakes/migrate_executor.go --fake-name MigrateExecutor . MigrateExecutor
type MigrateExecutor interface {
	Exec(db MigrationDb, dialect string, m migrate.MigrationSource, dir migrate.MigrationDirection) (int, error)
}

//go:generate counterfeiter -o fakes/migration_db.go --fake-name MigrationDb . MigrationDb
type MigrationDb interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	DriverName() string
}

var policyServerMigrations dbSpecificPolicyServerMigrations = dbSpecificPolicyServerMigrations{
	policyServerMigration{
		"1",
		SchemasV0Up,
		SchemasV0Down,
	},
	policyServerMigration{
		"2",
		SchemasV1Up,
		SchemasV1Down,
	},
	policyServerMigration{
		"3",
		SchemasV2Up,
		schemaDownNotImplemented,
	},
}

func PerformMigrations(driverName string, migrationDb MigrationDb, migrateExecutor MigrateExecutor) (int, error) {
	if !policyServerMigrations.supportsDatabase(driverName) {
		return 0, fmt.Errorf("unsupported driver: %s", driverName)
	}

	numMigrations, err := migrateExecutor.Exec(
		migrationDb,
		driverName,
		migrate.MemoryMigrationSource{
			Migrations: policyServerMigrations.asExecutorMigrations(driverName),
		},
		migrate.Up)

	if err != nil {
		return numMigrations, fmt.Errorf("executing migration: %s", err)
	}
	return numMigrations, nil
}

type dbSpecificPolicyServerMigrations []policyServerMigration
type policyServerMigration struct {
	Id   string
	Up   map[string][]string
	Down map[string][]string
}

func (psm *policyServerMigration) supportsDB(driverName string) bool {
	_, foundUp := psm.Up[driverName]
	_, foundDown := psm.Down[driverName]

	return foundUp && foundDown
}

func (psm *policyServerMigration) asMigration(driverName string) *migrate.Migration {
	return &migrate.Migration{
		Id:   psm.Id,
		Up:   psm.Up[driverName],
		Down: psm.Down[driverName],
	}
}

func (s dbSpecificPolicyServerMigrations) asExecutorMigrations(driverName string) []*migrate.Migration {
	migrationMapped := []*migrate.Migration{}

	for _, migration := range s {
		migrationMapped = append(migrationMapped, migration.asMigration(driverName))
	}
	return migrationMapped
}

func (s dbSpecificPolicyServerMigrations) supportsDatabase(driverName string) bool {
	for _, migration := range s {
		if !migration.supportsDB(driverName) {
			return false
		}
	}
	return true
}

var SchemasV0Up = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS groups (
		id int NOT NULL AUTO_INCREMENT,
		guid varchar(255),
		UNIQUE (guid),
		PRIMARY KEY (id)
	);`,
		`CREATE TABLE IF NOT EXISTS destinations (
		id int NOT NULL AUTO_INCREMENT,
		group_id int REFERENCES groups(id),
		port int,
		protocol varchar(255),
		UNIQUE (group_id, port, protocol),
		PRIMARY KEY (id)
	);`,
		`CREATE TABLE IF NOT EXISTS policies (
		id int NOT NULL AUTO_INCREMENT,
		group_id int REFERENCES groups(id),
		destination_id int REFERENCES destinations(id),
		UNIQUE (group_id, destination_id),
		PRIMARY KEY (id)
	);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS groups (
		id SERIAL PRIMARY KEY,
		guid text,
		UNIQUE (guid)
	);`,
		`CREATE TABLE IF NOT EXISTS destinations (
		id SERIAL PRIMARY KEY,
		group_id int REFERENCES groups(id),
		port int,
		protocol text,
		UNIQUE (group_id, port, protocol)
	);`,
		`CREATE TABLE IF NOT EXISTS policies (
		id SERIAL PRIMARY KEY,
		group_id int REFERENCES groups(id),
		destination_id int REFERENCES destinations(id),
		UNIQUE (group_id, destination_id)
	);`,
	},
}

var SchemasV0Down = map[string][]string{
	"mysql": {
		"DROP TABLE policies", "DROP TABLE destinations", "DROP TABLE groups",
	},
	"postgres": {
		"DROP TABLE policies", "DROP TABLE destinations", "DROP TABLE groups",
	},
}

var SchemasV1Up = map[string][]string{
	"mysql": {
		`ALTER TABLE destinations ADD COLUMN start_port int;`,
		`ALTER TABLE destinations ADD COLUMN end_port int;`,
		`UPDATE destinations SET start_port = port;`,
		`UPDATE destinations SET end_port = port;`,
	},
	"postgres": {
		`ALTER TABLE destinations ADD COLUMN start_port int;`,
		`ALTER TABLE destinations ADD COLUMN end_port int;`,
		`UPDATE destinations SET start_port = port;`,
		`UPDATE destinations SET end_port = port;`,
	},
}

var SchemasV1Down = map[string][]string{
	"mysql": {
		`ALTER TABLE destinations DROP COLUMN start_port;`,
		`ALTER TABLE destinations DROP COLUMN end_port;`,
	},
	"postgres": {
		`ALTER TABLE destinations DROP COLUMN start_port;`,
		`ALTER TABLE destinations DROP COLUMN end_port;`,
	},
}

var SchemasV2Up = map[string][]string{
	"mysql": {
		`alter table destinations drop index group_id`,
		`alter table destinations add unique key destinations_group_id_start_port_end_port_protocol_key (group_id, start_port, end_port, protocol)`,
	},
	"postgres": {
		`ALTER TABLE destinations
		 DROP CONSTRAINT destinations_group_id_port_protocol_key
         ,ADD CONSTRAINT destinations_group_id_start_port_end_port_protocol_key UNIQUE (group_id, start_port, end_port, protocol)`,
	},
}

var schemaDownNotImplemented = map[string][]string{
	"mysql": {
		``,
	},
	"postgres": {
		``,
	},
}
