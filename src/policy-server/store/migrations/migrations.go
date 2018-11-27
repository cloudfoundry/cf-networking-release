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
	PolicyServerMigration{
		Id: "19",
		Up: migration_v0019,
	},
	PolicyServerMigration{
		Id: "20",
		Up: migration_v0020,
	},
	PolicyServerMigration{
		Id: "21",
		Up: migration_v0021,
	},
	PolicyServerMigration{
		Id: "22",
		Up: migration_v0022,
	},
	PolicyServerMigration{
		Id: "23",
		Up: migration_v0023,
	},
	PolicyServerMigration{
		Id: "24",
		Up: migration_v0024,
	},
	PolicyServerMigration{
		Id: "25",
		Up: migration_v0025,
	},
	PolicyServerMigration{
		Id: "26",
		Up: migration_v0026,
	},
	PolicyServerMigration{
		Id: "27",
		Up: migration_v0027,
	},
	PolicyServerMigration{
		Id: "28",
		Up: migration_v0028,
	},
	PolicyServerMigration{
		Id: "29",
		Up: migration_v0029,
	},
	PolicyServerMigration{
		Id: "30",
		Up: migration_v0030,
	},
	PolicyServerMigration{
		Id: "31",
		Up: migration_v0031,
	},
	PolicyServerMigration{
		Id: "32",
		Up: migration_v0032,
	},
	PolicyServerMigration{
		Id: "33",
		Up: migration_v0033,
	},
	PolicyServerMigration{
		Id: "34",
		Up: migration_v0034,
	},
	PolicyServerMigration{
		Id: "35",
		Up: migration_v0035,
	},
	PolicyServerMigration{
		Id: "36",
		Up: migration_v0036,
	},
	PolicyServerMigration{
		Id: "37",
		Up: migration_v0037,
	},
	PolicyServerMigration{
		Id: "38",
		Up: migration_v0038,
	},
	PolicyServerMigration{
		Id: "39",
		Up: migration_v0039,
	},
	PolicyServerMigration{
		Id: "40",
		Up: migration_v0040,
	},
	PolicyServerMigration{
		Id: "41",
		Up: migration_v0041,
	},
	PolicyServerMigration{
		Id: "42",
		Up: migration_v0042,
	},
	PolicyServerMigration{
		Id: "43",
		Up: migration_v0043,
	},
	PolicyServerMigration{
		Id: "44",
		Up: migration_v0044,
	},
	PolicyServerMigration{
		Id: "45",
		Up: migration_v0045,
	},
	PolicyServerMigration{
		Id: "46",
		Up: migration_v0046,
	},
	PolicyServerMigration{
		Id: "47",
		Up: migration_v0047,
	},
	PolicyServerMigration{
		Id: "48",
		Up: migration_v0048,
	},
	PolicyServerMigration{
		Id: "49",
		Up: migration_v0049,
	},
	PolicyServerMigration{
		Id: "50",
		Up: migration_v0050,
	},
	PolicyServerMigration{
		Id: "51",
		Up: migration_v0051,
	},
	PolicyServerMigration{
		Id: "52",
		Up: migration_v0052,
	},
	PolicyServerMigration{
		Id: "53",
		Up: migration_v0053,
	},
	PolicyServerMigration{
		Id: "54",
		Up: migration_v0054,
	},
	PolicyServerMigration{
		Id: "55",
		Up: migration_v0055,
	},
	PolicyServerMigration{
		Id: "56",
		Up: migration_v0056,
	},
	PolicyServerMigration{
		Id: "57",
		Up: migration_v0057,
	},
}
