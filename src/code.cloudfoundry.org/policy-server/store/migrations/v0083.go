package migrations

var migration_v0083 = map[string][]string{
	"mysql": {
		`CREATE TABLE  IF NOT EXISTS running_spaces (
			id bigint NOT NULL AUTO_INCREMENT,
			security_group_guid varchar(36) NOT NULL,
			space_guid varchar(36) NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY running_group_space (security_group_guid, space_guid),
			KEY running_security_group_guid_fkey (security_group_guid),
			CONSTRAINT running_security_group_guid_fkey
				FOREIGN KEY (security_group_guid) REFERENCES security_groups (guid) ON DELETE CASCADE
		)`,
		// migrate data from old columns to new tables - how???
		// DROP old columns
	},
	"postgres": {
		`CREATE TABLE  IF NOT EXISTS running_spaces (
			id BIGSERIAL PRIMARY KEY,
			security_group_guid varchar(36) NOT NULL,
			space_guid varchar(36) NOT NULL,
			UNIQUE KEY running_group_space (security_group_guid, space_guid),
			KEY running_security_group_guid_fkey (security_group_guid),
			CONSTRAINT running_security_group_guid_fkey
				FOREIGN KEY (security_group_guid) REFERENCES security_groups (guid) ON DELETE CASCADE
		)`,
	},
}
