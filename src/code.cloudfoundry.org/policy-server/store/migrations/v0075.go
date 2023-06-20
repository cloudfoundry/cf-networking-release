package migrations

var migration_v0075 = map[string][]string{
	"mysql": []string{
		`ALTER TABLE security_groups MODIFY COLUMN id bigint;`,
	},
	"postgres": []string{
		`ALTER TABLE security_groups ALTER COLUMN id type bigint;`,
		`ALTER SEQUENCE public.security_groups_id_seq as bigint MAXVALUE 9223372036854775807;`,
	},
}
