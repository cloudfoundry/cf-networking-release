package migrations

var migration_v0015 = map[string][]string{
	"mysql": {
		`UPDATE ip_ranges SET end_port = 0;`,
	},
	"postgres": {
		`UPDATE ip_ranges SET end_port = 0;`,
	},
}
