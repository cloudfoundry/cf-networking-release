package migrations

var migration_v0024 = map[string][]string{
	"mysql": {
		`ALTER TABLE terminals MODIFY guid VARCHAR(36) NOT NULL UNIQUE;`,
	},
	"postgres": {
		`ALTER TABLE terminals ADD CONSTRAINT terminals_guid_unique UNIQUE (guid),
		 ALTER COLUMN guid SET NOT NULL;`,
	},
}
