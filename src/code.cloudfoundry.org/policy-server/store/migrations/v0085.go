package migrations

var migration_v0085 = map[string][]string{
	"mysql": {
		`CREATE INDEX global_running_idx ON security_groups (running_default)`,
	},
	"postgres": {},
}
