package migrations

var migration_v0021 = map[string][]string{
	"mysql": {},
	"postgres": {
		`CREATE INDEX metadata_name_idx ON destination_metadatas (name);`,
	},
}
