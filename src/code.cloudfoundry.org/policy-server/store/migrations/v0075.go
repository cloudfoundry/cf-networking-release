package migrations

var migration_v0075 = map[string][]string{
	"mysql": []string{
		`ALTER TABLE security_groups MODIFY COLUMN id bigint AUTO_INCREMENT;`,
	},
	"postgres": []string{
		`ALTER TABLE security_groups ALTER COLUMN id type bigint;`,
	},
}
