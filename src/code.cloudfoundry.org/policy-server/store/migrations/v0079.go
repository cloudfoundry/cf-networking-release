package migrations

// Adding first record of last updated field to policies
// set to current date

var migration_v0079 = map[string][]string{
	"mysql": {
		`ALTER TABLE policies_info MODIFY last_updated TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)`,
	},
	"postgres": {},
}
