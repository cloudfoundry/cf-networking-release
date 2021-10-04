package migrations

var migration_v0066 = map[string][]string{
	"mysql": {
		`DROP PROCEDURE drop_destination_index`,
	},
	"postgres": {},
}
