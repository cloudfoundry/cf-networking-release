package migrations

var migration_v0082 = map[string][]string{
	"mysql": {
		`CREATE INDEX staging_spaces_idx ON security_groups ((CAST(staging_spaces -> '$[*]' AS CHAR(36) ARRAY)))`,
	},
	"postgres": {},
}
