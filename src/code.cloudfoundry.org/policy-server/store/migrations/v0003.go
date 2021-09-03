package migrations

var migration_v0003 = map[string][]string{
	"mysql": {
		`ALTER TABLE groups ADD COLUMN type varchar(255) DEFAULT 'app'`,
		`CREATE INDEX idx_type ON groups (type)`,
	},

	"postgres": {
		`ALTER TABLE groups ADD COLUMN type text DEFAULT 'app'`,
		`CREATE INDEX idx_type ON groups (type)`,
	},
}

var migration_modified_v0003 = map[string][]string{
	"mysql": {
		`ALTER TABLE groups ADD COLUMN type varchar(255) DEFAULT 'app'`,
	},

	"postgres": {
		`ALTER TABLE groups ADD COLUMN type text DEFAULT 'app'`,
	},
}

var migration_modified_v0003a = map[string][]string{
	"mysql": {
		`CREATE INDEX idx_type ON groups (type)`,
	},

	"postgres": {
		`CREATE INDEX idx_type ON groups (type)`,
	},
}
