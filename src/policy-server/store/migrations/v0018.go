package migrations

var migration_v0018 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS spaces (
		id int NOT NULL AUTO_INCREMENT,
		PRIMARY KEY (id),
		terminal_id int,
		INDEX spaces_terminal_id_idx (terminal_id),
		CONSTRAINT spaces_terminal_id_fk
			FOREIGN KEY (terminal_id)
			REFERENCES terminals(id),
		space_guid varchar(255),
		UNIQUE(space_guid),
		INDEX spaces_space_guid_idx (space_guid)
	);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS spaces (
		id SERIAL PRIMARY KEY,
		terminal_id int,
		FOREIGN KEY (terminal_id) references terminals(id),
		space_guid text CONSTRAINT spaces_space_guid_unique UNIQUE
	);`,
	},
}
