package migrations

var migration_v0061 = map[string][]string{
	"mysql": {
		`CALL drop_terminal_guid_unique_index();`,
	},
	"postgres": {},
}
