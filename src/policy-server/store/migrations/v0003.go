package migrations

var migration_v0003 = map[string][]string{
	"mysql": {
		`ALTER TABLE groups ADD COLUMN type varchar(255) DEFAULT 'app'`,
		`CREATE INDEX idx_type ON groups (type)`,
	},
}
