package migrations

var migration_v0058 = map[string][]string{
	"mysql": {
		`ALTER TABLE egress_policies ADD CONSTRAINT egress_policies_all_columns_but_guid_unique UNIQUE (source_guid, destination_guid, app_lifecycle)`,
	},
	"postgres": {
		`ALTER TABLE egress_policies ADD CONSTRAINT egress_policies_all_columns_but_guid_unique UNIQUE (source_guid, destination_guid, app_lifecycle)`,
	},
}
