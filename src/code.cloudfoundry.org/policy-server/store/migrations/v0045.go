package migrations

var migration_v0045 = map[string][]string{
	"mysql": {
		`CREATE INDEX apps_terminal_guid_idx ON apps (terminal_guid);`,
	},
	"postgres": {
		`CREATE INDEX apps_terminal_guid_idx ON apps (terminal_guid);`,
	},
}
