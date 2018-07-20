package migrations

var migration_v0005 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS egress_policies (
		id int NOT NULL AUTO_INCREMENT,
		PRIMARY KEY (id),
		source_id int,
		INDEX source_id_idx (source_id),
		CONSTRAINT egress_policies_source_id_fk
            FOREIGN KEY (source_id)
			REFERENCES terminals(id),
		destination_id int,
		INDEX destination_id_idx (destination_id),
		CONSTRAINT egress_policies_destination_id_fk
            FOREIGN KEY (destination_id)
			REFERENCES terminals(id)
	);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS egress_policies (
		id SERIAL PRIMARY KEY,
		source_id int,
        FOREIGN KEY (source_id) references terminals(id),
		destination_id int,
        FOREIGN KEY (destination_id) references terminals(id)
	);`,
	},
}
