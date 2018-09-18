package migrations

var migration_v0056 = map[string][]string{
	"mysql": {
		`ALTER TABLE egress_policies ADD CONSTRAINT egress_policies_source_guid_destination_guid_unique UNIQUE (source_guid, destination_guid)`,
	},
	"postgres": {
		`ALTER TABLE egress_policies ADD CONSTRAINT egress_policies_source_guid_destination_guid_unique UNIQUE (source_guid, destination_guid)`,
	},
}
