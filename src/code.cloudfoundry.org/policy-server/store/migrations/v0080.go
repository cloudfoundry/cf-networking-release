package migrations

var migration_v0080 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS security_groups_info (
			id int NOT NULL AUTO_INCREMENT,
			PRIMARY KEY (id),
			last_updated TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
		);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS security_groups_info (
			id SERIAL PRIMARY KEY,
			last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	},
}
