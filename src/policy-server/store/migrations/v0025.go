package migrations

var migration_v0025 = map[string][]string{
	"mysql": {
		`ALTER TABLE apps ADD COLUMN terminal_guid VARCHAR(36);`,
	},
	"postgres": {
		`ALTER TABLE apps ADD COLUMN terminal_guid VARCHAR(36);`,
	},
}
