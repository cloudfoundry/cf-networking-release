package migrations

var migration_v0083 = map[string][]string{
	"mysql": {},
	// This ran into issues with index limits when ASGs are bound to > 148 spaces. Rolling it back.
	//	`CREATE INDEX running_spaces_idx ON security_groups ((CAST(running_spaces -> '$[*]' AS CHAR(36) ARRAY)))`,
	"postgres": {},
}
