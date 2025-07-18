package migrations

var migration_v0092 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS "running_security_groups_spaces" (
			space_guid VARCHAR(36) NOT NULL,
			security_group_guid VARCHAR(36) NOT NULL,
			UNIQUE (space_guid, security_group_guid),
			PRIMARY KEY (space_guid, security_group_guid),
			CONSTRAINT "running_security_groups_guid_fkey" FOREIGN KEY (security_group_guid) REFERENCES "security_groups" (guid) ON DELETE CASCADE
			)
		`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS running_security_groups_spaces (
			space_guid varchar(36) NOT NULL,
			security_group_guid varchar(36) NOT NULL
				CONSTRAINT running_sg_spaces_fk REFERENCES security_groups(guid) ON DELETE CASCADE,
			CONSTRAINT running_sg_spaces_unique UNIQUE (space_guid, security_group_guid),
			CONSTRAINT running_sg_spaces_pk PRIMARY KEY (space_guid, security_group_guid)
			)
		`,
	},
}
