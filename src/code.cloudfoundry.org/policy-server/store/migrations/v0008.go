package migrations

var migration_v0008 = map[string][]string{
	"mysql": {},
	"postgres": {
		`CREATE INDEX source_terminal_id_idx ON egress_policies (source_id);`,
	},
}
