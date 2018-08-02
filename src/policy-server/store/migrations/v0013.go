package migrations

var migration_v0013 = map[string][]string{
	"mysql": {
		`ALTER TABLE ip_ranges ADD COLUMN end_port int;`,
	},
	"postgres": {
		`ALTER TABLE ip_ranges ADD COLUMN end_port int;`,
	},
}
