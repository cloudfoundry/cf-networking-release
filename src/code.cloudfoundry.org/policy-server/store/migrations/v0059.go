package migrations

var migration_v0059 = map[string][]string{
	"mysql": {
		`ALTER TABLE egress_policies DROP INDEX egress_policies_source_guid_destination_guid_unique`,
	},
	"postgres": {
		`ALTER TABLE egress_policies DROP CONSTRAINT egress_policies_source_guid_destination_guid_unique`,
	},
}
