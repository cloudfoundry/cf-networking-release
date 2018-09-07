package migrations

var migration_v0049 = map[string][]string{
	"mysql": {
		`CREATE INDEX egress_policies_source_guid_idx ON egress_policies (source_guid);`,
	},
	"postgres": {
		`CREATE INDEX egress_policies_source_guid_idx ON egress_policies (source_guid);`,
	},
}
