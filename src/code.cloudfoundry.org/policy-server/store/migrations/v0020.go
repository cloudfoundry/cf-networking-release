package migrations

var migration_v0020 = map[string][]string{
	"mysql": {},
	"postgres": {
		`CREATE INDEX metadata_terminal_id_idx ON destination_metadatas (terminal_id);`,
	},
}
