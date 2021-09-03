package migrations

var migration_v0044 = map[string][]string{
	"mysql": {
		`ALTER TABLE terminals
		 DROP id;`,
	},
	"postgres": {},
}
