package migrations

var MigrationsToPerform policyServerMigrations = policyServerMigrations{
	policyServerMigration{
		"1",
		migration_v0001,
	},
	policyServerMigration{
		"2",
		migration_v0002,
	},
}
