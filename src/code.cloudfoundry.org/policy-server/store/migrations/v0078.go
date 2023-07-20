package migrations

// Adding first record of last updated field to policies
// set to current date

var migration_v0078 = map[string][]string{
	"mysql": {
		`INSERT INTO policies_info (last_updated) VALUES (CURRENT_TIMESTAMP);`,
	},
	"postgres": {
		`INSERT INTO policies_info (last_updated) VALUES (CURRENT_TIMESTAMP);`,
	},
}
