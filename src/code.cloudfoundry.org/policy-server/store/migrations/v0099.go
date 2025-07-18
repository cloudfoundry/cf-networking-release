package migrations

var migration_v0099 = map[string][]string{
	"mysql": {
		`DROP TABLE IF EXISTS temp_sequence`,
	},
	"postgres": {},
}
