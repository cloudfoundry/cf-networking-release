package migrations

var migration_v0091 = map[string][]string{
	"mysql": {
		`DROP PROCEDURE drop_staging_spaces_index;`,
	},
	"postgres": {},
}

var migration_v0092 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS spaces (
			id int NOT NULL AUTO_INCREMENT,
			guid varchar(255),
			lastUpdated TIMESTAMP,
			asgs LONGTEXT,
			UNIQUE (guid),
			PRIMARY KEY (id)
		);`,
	},

	"postgres": {
		`CREATE TABLE IF NOT EXISTS spaces (
			id int NOT NULL AUTO_INCREMENT,
			guid varchar(255),
			lastUpdated TIMESTAMP,
			asgs LONGTEXT,
			UNIQUE (guid),
			PRIMARY KEY (id)
		);`,
	},
}

var migration_v0093 = map[string][]string{
	// FIXME: move this up to v92 for the final product
	"mysql": {
		`ALTER TABLE spaces ADD COLUMN hash VARCHAR(255);`,
	},
}
