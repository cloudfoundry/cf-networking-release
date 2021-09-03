package migrations

var migration_v0035 = map[string][]string{
	"mysql": {
		`UPDATE destination_metadatas
		 SET terminal_guid = terminal_id;`,
	},
	"postgres": {
		`UPDATE destination_metadatas
		 SET terminal_guid = terminal_id;`,
	},
}
