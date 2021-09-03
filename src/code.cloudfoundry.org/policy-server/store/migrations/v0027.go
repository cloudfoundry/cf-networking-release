package migrations

var migration_v0027 = map[string][]string{
	"mysql": {
		`ALTER TABLE apps
		 ADD CONSTRAINT apps_terminal_guid_fk FOREIGN KEY (terminal_guid)
		 REFERENCES terminals(guid),
		 MODIFY terminal_guid VARCHAR(36) NOT NULL UNIQUE,
		 DROP FOREIGN KEY apps_terminal_id_fk,
		 DROP terminal_id;`,
	},
	"postgres": {
		`ALTER TABLE apps
		 ADD CONSTRAINT apps_terminal_guid_fk FOREIGN KEY (terminal_guid) REFERENCES terminals(guid),
		 ADD CONSTRAINT apps_terminal_guid_unique UNIQUE (terminal_guid),
		 ALTER COLUMN terminal_guid SET NOT NULL,
		 DROP terminal_id;`,
	},
}
