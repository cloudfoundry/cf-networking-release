package migrations

// Adding policies information table to store the date
// when policies were last updated

var migration_v0077 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS policies_info (
			id int NOT NULL AUTO_INCREMENT,
			PRIMARY KEY (id),
			last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS policies_info (
			id SERIAL PRIMARY KEY,
			last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	},
}
