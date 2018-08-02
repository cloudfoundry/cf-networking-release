package migrations

var migration_v0012 = map[string][]string{
	"mysql": {
		`ALTER TABLE ip_ranges ADD COLUMN start_port int;`,
	},
	"postgres": {
		`ALTER TABLE ip_ranges ADD COLUMN start_port int;`,
	},
}
