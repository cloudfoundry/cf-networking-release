package migrations

var migration_v0088 = map[string][]string{
	"mysql": {
		`CALL drop_running_spaces_index();`,
	},
	"postgres": {},
}
