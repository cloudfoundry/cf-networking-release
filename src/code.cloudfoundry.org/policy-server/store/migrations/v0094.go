package migrations

var migration_v0094 = map[string][]string{
	"mysql": {
		`ALTER TABLE security_groups ADD COLUMN hash VARCHAR(255) DEFAULT ''`,
	},
	"postgres": {
		`ALTER TABLE security_groups ADD COLUMN hash VARCHAR(255) DEFAULT ''`,
	},
}
