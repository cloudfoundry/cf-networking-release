package migrations

var migration_v0063 = map[string][]string{
	"mysql": {
		`DROP PROCEDURE drop_terminal_guid_unique_index`,
	},
	"postgres": {},
}
