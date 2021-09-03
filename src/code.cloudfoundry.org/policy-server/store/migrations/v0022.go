package migrations

var migration_v0022 = map[string][]string{
	"mysql": {
		`ALTER TABLE terminals ADD COLUMN guid VARCHAR(36);`,
	},
	"postgres": {
		`ALTER TABLE terminals ADD COLUMN guid VARCHAR(36);`,
	},
}
