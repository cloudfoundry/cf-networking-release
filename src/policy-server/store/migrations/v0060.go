package migrations

var migration_v0060 = map[string][]string{
	"mysql": {
		`CREATE TABLE defaults (
			id INT NOT NULL PRIMARY KEY AUTO_INCREMENT,
			terminal_guid VARCHAR(36) NOT NULL,
			FOREIGN KEY fk_terminal_default(terminal_guid) REFERENCES terminals(guid),
			CONSTRAINT default_unique_guid UNIQUE(terminal_guid)
		)`,
	},
	"postgres": {
		`CREATE TABLE defaults (
			id SERIAL,
			terminal_guid VARCHAR(36)
										NOT NULL
										CONSTRAINT default_unique_guid UNIQUE
										CONSTRAINT fk_terminal_default REFERENCES terminals(guid)
		)`,
	},
}
