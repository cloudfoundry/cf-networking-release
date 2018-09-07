package migrations

var migration_v0037 = map[string][]string{
	"mysql": {
		`ALTER TABLE egress_policies ADD COLUMN source_guid VARCHAR(36);`,
	},
	"postgres": {
		`ALTER TABLE egress_policies ADD COLUMN source_guid VARCHAR(36);`,
	},
}
