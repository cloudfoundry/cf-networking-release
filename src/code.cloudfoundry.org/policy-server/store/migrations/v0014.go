package migrations

var migration_v0014 = map[string][]string{
	"mysql": {
		`UPDATE ip_ranges SET start_port = 0;`,
	},
	"postgres": {
		`UPDATE ip_ranges SET start_port = 0;`,
	},
}
