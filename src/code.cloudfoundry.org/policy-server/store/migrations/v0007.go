package migrations

var migration_v0007 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS apps (
		id int NOT NULL AUTO_INCREMENT,
		PRIMARY KEY (id),
		terminal_id int,
		INDEX apps_terminal_id_idx (terminal_id),
		CONSTRAINT apps_terminal_id_fk 
			FOREIGN KEY (terminal_id)
			REFERENCES terminals(id),
		app_guid varchar(255),
		UNIQUE(app_guid),
		INDEX apps_app_guid_idx (app_guid)
	);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS apps (
		id SERIAL PRIMARY KEY,
		terminal_id int,
		FOREIGN KEY (terminal_id) references terminals(id),
		app_guid text CONSTRAINT apps_app_guid_unique UNIQUE
	);`,
	},
}
