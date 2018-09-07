package migrations

var migration_v0050 = map[string][]string{
	"mysql": {
		`CREATE INDEX egress_policies_destination_guid_idx ON egress_policies (destination_guid);`,
	},
	"postgres": {
		`CREATE INDEX egress_policies_destination_guid_idx ON egress_policies (destination_guid);`,
	},
}
