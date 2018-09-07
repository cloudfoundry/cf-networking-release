package migrations

var migration_v0046 = map[string][]string{
	"mysql": {
		`CREATE INDEX spaces_terminal_guid_idx ON spaces (terminal_guid);`,
	},
	"postgres": {
		`CREATE INDEX spaces_terminal_guid_idx ON spaces (terminal_guid);`,
	},
}
