package migrations

var migration_v0096 = map[string][]string{
	"mysql": {
		`CALL generate_sequence();`,
	},
	"postgres": {},
}
