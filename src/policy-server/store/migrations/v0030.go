package migrations

var migration_v0030 = map[string][]string{
	"mysql": {
		`ALTER TABLE spaces
		 ADD CONSTRAINT spaces_terminal_guid_fk FOREIGN KEY (terminal_guid)
		 REFERENCES terminals(guid),
		 MODIFY terminal_guid VARCHAR(36) NOT NULL UNIQUE,
		 DROP FOREIGN KEY spaces_terminal_id_fk,
		 DROP terminal_id;`,
	},
	"postgres": {
		`ALTER TABLE spaces
		 ADD CONSTRAINT spaces_terminal_guid_fk FOREIGN KEY (terminal_guid) REFERENCES terminals(guid),
		 ADD CONSTRAINT spaces_terminal_guid_unique UNIQUE (terminal_guid),
		 ALTER COLUMN terminal_guid SET NOT NULL,
		 DROP terminal_id;`,
	},
}
