package migrations

var migration_v0082 = map[string][]string{
	"mysql": {},
	// This ran into issues with index limits when ASGs are bound to > 148 spaces. Rolling it back.
	//		`CREATE INDEX staging_spaces_idx ON security_groups ((CAST(staging_spaces -> '$[*]' AS CHAR(36) ARRAY)))`,
	"postgres": {},
}
