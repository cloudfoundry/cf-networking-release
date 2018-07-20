package migrations

var migration_v0009 = map[string][]string{
	"mysql": {},
	"postgres": {
		`CREATE INDEX destination_terminal_id_idx ON egress_policies (destination_id);`,
	},
}
