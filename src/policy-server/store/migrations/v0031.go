package migrations

var migration_v0031 = map[string][]string{
	"mysql": {
		`ALTER TABLE ip_ranges ADD COLUMN terminal_guid VARCHAR(36);`,
	},
	"postgres": {
		`ALTER TABLE ip_ranges ADD COLUMN terminal_guid VARCHAR(36);`,
	},
}
