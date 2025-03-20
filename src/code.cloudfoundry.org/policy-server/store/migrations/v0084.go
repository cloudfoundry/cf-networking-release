package migrations

var migration_v0084 = map[string][]string{
	"mysql": {
		`CREATE INDEX global_staging_idx ON security_groups (staging_default)`,
	},
	"postgres": {},
}
