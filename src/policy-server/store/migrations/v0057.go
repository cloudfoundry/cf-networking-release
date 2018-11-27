package migrations

var migration_v0057 = map[string][]string{
	"mysql": {
		`ALTER TABLE egress_policies ADD app_lifecycle VARCHAR(16) DEFAULT 'all'`,
	},
	"postgres": {
		`ALTER TABLE egress_policies ADD app_lifecycle VARCHAR(16) DEFAULT 'all'`,
	},
}
