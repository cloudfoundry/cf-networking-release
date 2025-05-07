package migrations

var migration_v0091 = map[string][]string{
	"mysql": {
		`DROP PROCEDURE drop_staging_spaces_index;`,
	},
	"postgres": {},
}
