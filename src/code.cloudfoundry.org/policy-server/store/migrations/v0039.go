package migrations

var migration_v0039 = map[string][]string{
	"mysql": {
		`ALTER TABLE egress_policies
		 ADD CONSTRAINT egress_policies_source_guid_fk FOREIGN KEY (source_guid)
		 REFERENCES terminals(guid),
		 MODIFY source_guid VARCHAR(36) NOT NULL,
		 DROP FOREIGN KEY egress_policies_source_id_fk,
		 DROP source_id;`,
	},
	"postgres": {
		`ALTER TABLE egress_policies
		 ADD CONSTRAINT egress_policies_source_guid_fk FOREIGN KEY (source_guid) REFERENCES terminals(guid),
		 ALTER COLUMN source_guid SET NOT NULL,
		 DROP source_id;`,
	},
}
