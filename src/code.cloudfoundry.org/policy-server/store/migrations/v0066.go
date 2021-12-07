package migrations

var migration_v0066 = map[string][]string{
	"mysql": {
		`DROP PROCEDURE IF EXISTS drop_destination_index`,
	},
	"postgres": {},
}
