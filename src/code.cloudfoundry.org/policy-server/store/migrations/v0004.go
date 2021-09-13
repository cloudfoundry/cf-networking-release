package migrations

var migration_v0004 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS terminals (
		id int NOT NULL AUTO_INCREMENT,
		PRIMARY KEY (id)
	);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS terminals (
		id SERIAL PRIMARY KEY
	);`,
	},
}
