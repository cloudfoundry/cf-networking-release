package migrations

var migration_v0032 = map[string][]string{
	"mysql": {
		`UPDATE ip_ranges
		 SET terminal_guid = terminal_id;`,
	},
	"postgres": {
		`UPDATE ip_ranges
		 SET terminal_guid = terminal_id;`,
	},
}
