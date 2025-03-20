package migrations

var migration_v0083 = map[string][]string{
	"mysql": {
		`CREATE INDEX running_spaces_idx ON security_groups ((CAST(running_spaces -> '$[*]' AS CHAR(36) ARRAY)))`,
	},
	"postgres": {},
}
