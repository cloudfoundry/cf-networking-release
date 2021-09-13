package migrations

var migration_v0006 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS ip_ranges (
		id int NOT NULL AUTO_INCREMENT,
		PRIMARY KEY (id),
		protocol varchar(255),
		start_ip varchar(255), 
		end_ip varchar(255),
		terminal_id int,
		INDEX ip_range_terminal_id_idx (terminal_id),
		CONSTRAINT ip_range_terminal_id_fk
            FOREIGN KEY (terminal_id)
			REFERENCES terminals(id)
	);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS ip_ranges (
		id SERIAL PRIMARY KEY,
		protocol text,
		start_ip text,
		end_ip text,
		terminal_id int,
        FOREIGN KEY (terminal_id) references terminals(id)
	);`,
	},
}
