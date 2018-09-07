package migrations

var migration_v0029 = map[string][]string{
	"mysql": {
		`UPDATE spaces
		 SET terminal_guid = terminal_id;`,
	},
	"postgres": {
		`UPDATE spaces
		 SET terminal_guid = terminal_id;`,
	},
}
