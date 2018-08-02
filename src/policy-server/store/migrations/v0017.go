package migrations

var migration_v0017 = map[string][]string{
	"mysql": {
		`ALTER TABLE ip_ranges ADD COLUMN icmp_code INT DEFAULT 0;`,
	},
	"postgres": {
		`ALTER TABLE ip_ranges ADD COLUMN icmp_code INT DEFAULT 0;`,
	},
}
