package migrations

import (
	"database/sql"
	"fmt"

	"github.com/rubenv/sql-migrate"
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
}

type Migrator struct {
	MigrateAdapter migrateAdapter
}

func New() *Migrator {
	return &Migrator{
		MigrateAdapter: &MigrateAdapter{},
	}
}

func (m *Migrator) PerformMigrations(driverName string, migrationDb MigrationDb, maxNumMigrations int) (int, error) {
	if !policyServerMigrations.supportsDatabase(driverName) {
		return 0, fmt.Errorf("unsupported driver: %s", driverName)
	}

	numMigrations, err := m.MigrateAdapter.ExecMax(
		migrationDb,
		driverName,
		migrate.MemoryMigrationSource{
			Migrations: policyServerMigrations.asExecutorMigrations(driverName),
		},
		migrate.Up,
		maxNumMigrations,
	)

	if err != nil {
		return numMigrations, fmt.Errorf("executing migration: %s", err)
	}
	return numMigrations, nil
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
		`CREATE PROCEDURE drop_destination_index()
BEGIN
 SELECT CONSTRAINT_NAME INTO @name
 FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
 WHERE TABLE_NAME='destinations' AND COLUMN_NAME= 'port';

 SET @query = CONCAT('ALTER TABLE destinations DROP INDEX ', @name);

 PREPARE stmt FROM @query;

 EXECUTE stmt;

 DEALLOCATE PREPARE stmt;
 SET @query = NULL;
 SET @name = NULL;

END;`,
		`CALL drop_destination_index();`,
		`ALTER TABLE destinations ADD UNIQUE key unique_destination (group_id, start_port, end_port, protocol);`,
	},
	"postgres": {
		`ALTER TABLE destinations ADD COLUMN start_port int;`,
		`ALTER TABLE destinations ADD COLUMN end_port int;`,
		`UPDATE destinations SET start_port = port;`,
		`UPDATE destinations SET end_port = port;`,
		`DO $$DECLARE r record;
		 	BEGIN
		 		FOR r in select CONSTRAINT_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE WHERE TABLE_NAME='destinations' AND COLUMN_NAME='port'
		 		LOOP
		 			EXECUTE 'ALTER TABLE destinations DROP CONSTRAINT ' || quote_ident(r.CONSTRAINT_NAME);
		 		END LOOP;
		 	END$$;
	`,
		`ALTER TABLE destinations ADD CONSTRAINT unique_destination UNIQUE (group_id, start_port, end_port, protocol);`,
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

var schemaDownNotImplemented = map[string][]string{
	"mysql": {
		``,
	},
	"postgres": {
		``,
	},
}
