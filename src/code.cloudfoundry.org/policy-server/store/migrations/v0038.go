package migrations

var migration_v0038 = map[string][]string{
	"mysql": {
		`UPDATE egress_policies
		 SET source_guid = source_id;`,
	},
	"postgres": {
		`UPDATE egress_policies
		 SET source_guid = source_id;`,
	},
}
