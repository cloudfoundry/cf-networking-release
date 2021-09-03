package migrations

var migration_v0026 = map[string][]string{
	"mysql": {
		`UPDATE apps
		 SET terminal_guid = terminal_id;`,
	},
	"postgres": {
		`UPDATE apps
		 SET terminal_guid = terminal_id;`,
	},
}
