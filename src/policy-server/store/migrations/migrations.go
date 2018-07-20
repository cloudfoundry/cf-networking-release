package migrations

var MigrationsToPerform = PolicyServerMigrations{
	PolicyServerMigration{
		"1",
		migration_v0001,
	},
	PolicyServerMigration{
		"2",
		migration_v0002,
	},
	PolicyServerMigration{
		"3",
		migration_v0003,
	},
}
