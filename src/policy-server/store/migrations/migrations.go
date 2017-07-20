package migrations

var migrationsToPerform policyServerMigrations = policyServerMigrations{
	policyServerMigration{
		"1",
		migration_v0001,
		migrationDownNotImplemented,
	},
	policyServerMigration{
		"2",
		migration_v0002,
		migrationDownNotImplemented,
	},
}
