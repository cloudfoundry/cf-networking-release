package migrations

var migration_v0034 = map[string][]string{
	"mysql": {
		`ALTER TABLE destination_metadatas ADD COLUMN terminal_guid VARCHAR(36);`,
	},
	"postgres": {
		`ALTER TABLE destination_metadatas ADD COLUMN terminal_guid VARCHAR(36);`,
	},
}
