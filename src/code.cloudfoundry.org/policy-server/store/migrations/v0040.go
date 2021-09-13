package migrations

var migration_v0040 = map[string][]string{
	"mysql": {
		`ALTER TABLE egress_policies ADD COLUMN destination_guid VARCHAR(36);`,
	},
	"postgres": {
		`ALTER TABLE egress_policies ADD COLUMN destination_guid VARCHAR(36);`,
	},
}
