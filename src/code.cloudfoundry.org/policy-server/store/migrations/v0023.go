package migrations

var migration_v0023 = map[string][]string{
	"mysql": {
		`UPDATE terminals SET guid = id;`,
	},
	"postgres": {
		`UPDATE terminals SET guid = id;`,
	},
}
