package migrations

var migration_v0019 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS destination_metadatas (
		id int NOT NULL AUTO_INCREMENT,
		PRIMARY KEY (id),
		terminal_id int,
		INDEX metadata_terminal_id_idx (terminal_id),
		CONSTRAINT metadata_terminal_id_fk
			FOREIGN KEY (terminal_id)
			REFERENCES terminals(id),
		name nvarchar(255),
		description longtext,
		UNIQUE(name),
		INDEX metadata_name_idx (name)
	);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS destination_metadatas (
		id SERIAL PRIMARY KEY,
		terminal_id int,
		FOREIGN KEY (terminal_id) references terminals(id),
		name text CONSTRAINT metadata_name_unique UNIQUE,
		description text
	);`,
	},
}
