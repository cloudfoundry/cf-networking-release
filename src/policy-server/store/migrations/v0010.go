package migrations

var migration_v0010 = map[string][]string{
	"mysql": {},
	"postgres": {
		`CREATE INDEX ip_range_terminal_id_idx ON ip_ranges (terminal_id);`,
	},
}
