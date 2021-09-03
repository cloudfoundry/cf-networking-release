package migrations

var migration_v0048 = map[string][]string{
	"mysql": {
		`CREATE INDEX destination_metadatas_terminal_guid_idx ON destination_metadatas (terminal_guid);`,
	},
	"postgres": {
		`CREATE INDEX destination_metadatas_terminal_guid_idx ON destination_metadatas (terminal_guid);`,
	},
}
