package migrations

var migration_v0065 = map[string][]string{
	"mysql": {
		`ALTER TABLE policies
		ADD CONSTRAINT policies_destination_id_fkey
		FOREIGN KEY (destination_id)
		REFERENCES destinations(id);`,
	},
	"postgres": {},
}
