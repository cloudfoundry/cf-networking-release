package migrations

var migration_v0051 = map[string][]string{
	"mysql": {
		`ALTER TABLE egress_policies ADD COLUMN guid VARCHAR(36)`,
	},
	"postgres": {
		`ALTER TABLE egress_policies ADD COLUMN guid VARCHAR(36)`,
	},
}
