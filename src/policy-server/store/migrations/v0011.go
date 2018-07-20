package migrations

var migration_v0011 = map[string][]string{
	"mysql": {},
	"postgres": {
		`CREATE INDEX app_terminal_id_idx ON apps (terminal_id);`,
	},
}
