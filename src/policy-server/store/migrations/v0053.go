package migrations

var migration_v0053 = map[string][]string{
	"mysql": {
		`ALTER TABLE egress_policies MODIFY guid VARCHAR(36) NOT NULL UNIQUE;`,
	},
	"postgres": {
		`ALTER TABLE egress_policies ADD CONSTRAINT egress_policies_guid_unique UNIQUE (guid),
		 ALTER COLUMN guid SET NOT NULL;`,
	},
}
