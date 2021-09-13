package migrations

var migration_v0052 = map[string][]string{
	"mysql": {
		`UPDATE egress_policies SET guid = id;`,
	},
	"postgres": {
		`UPDATE egress_policies SET guid = id;`,
	},
}
