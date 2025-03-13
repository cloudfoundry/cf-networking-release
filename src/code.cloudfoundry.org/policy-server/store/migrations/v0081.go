package migrations

var migration_v0081 = map[string][]string{
	"mysql": {
		`INSERT INTO security_groups_info (last_updated) VALUES (CURRENT_TIMESTAMP(6));`,
	},
	"postgres": {
		`INSERT INTO security_groups_info (last_updated) VALUES (CURRENT_TIMESTAMP);`,
	},
}
