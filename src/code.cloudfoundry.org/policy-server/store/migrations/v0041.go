package migrations

var migration_v0041 = map[string][]string{
	"mysql": {
		`UPDATE egress_policies
		 SET destination_guid = destination_id;`,
	},
	"postgres": {
		`UPDATE egress_policies
		 SET destination_guid = destination_id;`,
	},
}
