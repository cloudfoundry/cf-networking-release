package migrations

var migration_v0090 = map[string][]string{
	"mysql": {
		`DROP PROCEDURE drop_running_spaces_index;`,
	},
	"postgres": {},
}
