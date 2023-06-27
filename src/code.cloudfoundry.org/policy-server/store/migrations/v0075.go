package migrations

var migration_v0075 = map[string][]string{
	"mysql": []string{
		`ALTER TABLE security_groups DROP FOREIGN KEY security_groups_pkey;`,
		`ALTER TABLE security_groups ALTER COLUMN id type bigint;`,
		`ALTER TABLE security_groups ADD PRIMARY KEY (id);`,
	},
	"postgres": []string{
		`ALTER TABLE security_groups DROP CONSTRAINT security_groups_pkey;`,
		`ALTER TABLE security_groups ALTER COLUMN id type bigint;`,
		`ALTER TABLE security_groups ADD PRIMARY KEY (id);`,
		`ALTER SEQUENCE public.security_groups_id_seq as bigint MAXVALUE 9223372036854775807;`,
	},
}
