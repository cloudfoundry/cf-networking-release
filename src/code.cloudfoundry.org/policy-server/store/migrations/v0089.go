package migrations

var migration_v0089 = map[string][]string{
	"mysql": {
		`CALL drop_staging_spaces_index();`,
	},
	"postgres": {},
}
