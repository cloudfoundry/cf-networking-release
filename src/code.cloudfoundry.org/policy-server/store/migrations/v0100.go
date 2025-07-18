package migrations

var migration_v0100 = map[string][]string{
	"mysql": {
		`DROP PROCEDURE IF EXISTS generate_sequence`,
	},
	"postgres": {},
}
