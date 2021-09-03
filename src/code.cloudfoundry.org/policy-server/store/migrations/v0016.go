package migrations

var migration_v0016 = map[string][]string{
	"mysql": {
		`ALTER TABLE ip_ranges ADD COLUMN icmp_type INT DEFAULT 0;`,
	},
	"postgres": {
		`ALTER TABLE ip_ranges ADD COLUMN icmp_type INT DEFAULT 0;`,
	},
}
