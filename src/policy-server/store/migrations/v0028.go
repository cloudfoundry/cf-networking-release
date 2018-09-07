package migrations

var migration_v0028 = map[string][]string{
	"mysql": {
		`ALTER TABLE spaces ADD COLUMN terminal_guid VARCHAR(36);`,
	},
	"postgres": {
		`ALTER TABLE spaces ADD COLUMN terminal_guid VARCHAR(36);`,
	},
}
