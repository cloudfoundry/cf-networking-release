package migrations

var migration_v0055 = map[string][]string{
	"mysql": {
		`ALTER TABLE egress_policies
		 DROP id;`,
	},
	"postgres": {},
}
