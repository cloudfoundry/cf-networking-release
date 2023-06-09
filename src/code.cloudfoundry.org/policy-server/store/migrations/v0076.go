package migrations

var migration_v0076 = map[string][]string{
	"mysql": []string{},
	"postgres": []string{
		`ALTER SEQUENCE security_groups_id_seq as bigint MAXVALUE 9223372036854775807;`,
	},
}
