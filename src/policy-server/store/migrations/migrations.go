package migrations

var empty_migration = map[string][]string{
	"mysql":    {},
	"postgres": {},
}

var V1LegacyMigrationsToPerform = PolicyServerMigrations{
	PolicyServerMigration{
		Id: "1",
		Up: migration_v0001,
	},
	PolicyServerMigration{
		Id: "1a",
		Up: empty_migration,
	},
	PolicyServerMigration{
		Id: "1b",
		Up: empty_migration,
	},
}

var V1ModifiedMigrationsToPerform = PolicyServerMigrations{
	PolicyServerMigration{
		Id: "1",
		Up: migration_modified_v0001,
	},
	PolicyServerMigration{
		Id: "1a",
		Up: migration_modified_v0001a,
	},
	PolicyServerMigration{
		Id: "1b",
		Up: migration_modified_v0001b,
	},
}

var V2LegacyMigrationsToPerform = PolicyServerMigrations{
	PolicyServerMigration{
		Id: "2",
		Up: migration_v0002,
	},
	PolicyServerMigration{
		Id: "2a",
		Up: empty_migration,
	},
	PolicyServerMigration{
		Id: "2b",
		Up: empty_migration,
	},
	PolicyServerMigration{
		Id: "2c",
		Up: empty_migration,
	},
	PolicyServerMigration{
		Id: "2d",
		Up: empty_migration,
	},
	PolicyServerMigration{
		Id: "2e",
		Up: empty_migration,
	},
	PolicyServerMigration{
		Id: "2f",
		Up: empty_migration,
	},
}

var V2ModifiedMigrationsToPerform = PolicyServerMigrations{
	PolicyServerMigration{
		Id: "2",
		Up: migration_modified_v0002,
	},
	PolicyServerMigration{
		Id: "2a",
		Up: migration_modified_v0002a,
	},
	PolicyServerMigration{
		Id: "2b",
		Up: migration_modified_v0002b,
	},
	PolicyServerMigration{
		Id: "2c",
		Up: migration_modified_v0002c,
	},
	PolicyServerMigration{
		Id: "2d",
		Up: migration_modified_v0002d,
	},
	PolicyServerMigration{
		Id: "2e",
		Up: migration_modified_v0002e,
	},
	PolicyServerMigration{
		Id: "2f",
		Up: migration_modified_v0002f,
	},
}

var V3LegacyMigrationsToPerform = PolicyServerMigrations{
	PolicyServerMigration{
		Id: "3",
		Up: migration_v0003,
	},
	PolicyServerMigration{
		Id: "3a",
		Up: empty_migration,
	},
}

var V3ModifiedMigrationsToPerform = PolicyServerMigrations{
	PolicyServerMigration{
		Id: "3",
		Up: migration_modified_v0003,
	},
	PolicyServerMigration{
		Id: "3a",
		Up: migration_modified_v0003a,
	},
}

var MigrationsToPerform = PolicyServerMigrations{
	PolicyServerMigration{
		Id: "4",
		Up: migration_v0004,
	},
	PolicyServerMigration{
		Id: "5",
		Up: migration_v0005,
	},
	PolicyServerMigration{
		Id: "6",
		Up: migration_v0006,
	},
	PolicyServerMigration{
		Id: "7",
		Up: migration_v0007,
	},
	PolicyServerMigration{
		Id: "8",
		Up: migration_v0008,
	},
	PolicyServerMigration{
		Id: "9",
		Up: migration_v0009,
	},
	PolicyServerMigration{
		Id: "10",
		Up: migration_v0010,
	},
	PolicyServerMigration{
		Id: "11",
		Up: migration_v0011,
	},
	PolicyServerMigration{
		Id: "12",
		Up: migration_v0012,
	},
	PolicyServerMigration{
		Id: "13",
		Up: migration_v0013,
	},
	PolicyServerMigration{
		Id: "14",
		Up: migration_v0014,
	},
	PolicyServerMigration{
		Id: "15",
		Up: migration_v0015,
	},
	PolicyServerMigration{
		Id: "16",
		Up: migration_v0016,
	},
	PolicyServerMigration{
		Id: "17",
		Up: migration_v0017,
	},
	PolicyServerMigration{
		Id: "18",
		Up: migration_v0018,
	},
}
