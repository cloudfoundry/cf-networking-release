package migrations

// These all use different migrations vs multiple statements in a text block,
// or multiple text blocks per migration, because of errors encountered
// with the migrator not running migrations in a transaction in mysql properly
// See commit ID f1280ed0eee131aa57bd9af0cdbc590fb36be80a for more information

var migration_v0067a = map[string][]string{
	"mysql": []string{
		`CREATE TABLE IF NOT EXISTS "security_groups" (
			id integer NOT NULL AUTO_INCREMENT PRIMARY KEY,
			guid varchar(36) NOT NULL,
			name varchar(255) NOT NULL,
			rules mediumtext,
			staging_default bool DEFAULT false,
			running_default bool DEFAULT false,
			UNIQUE (guid)
		);`,
	},
	"postgres": []string{
		`CREATE TABLE IF NOT EXISTS "security_groups" (
			id SERIAL PRIMARY KEY,
			guid varchar(36) NOT NULL,
 			name varchar(255) NOT NULL,
			rules text,
			staging_default bool DEFAULT false,
			running_default bool DEFAULT false,
			UNIQUE (guid)
		);`,
	},
}

var migration_v0067b = map[string][]string{
	"mysql": []string{
		`CREATE TABLE IF NOT EXISTS "staging_security_groups_spaces" (
			space_guid varchar(36) NOT NULL,
			security_group_guid varchar(36) NOT NULL,
			CONSTRAINT staging_security_groups_spaces_id_fk
				FOREIGN KEY (security_group_guid)
				REFERENCES security_groups(guid),
			PRIMARY KEY (space_guid, security_group_guid)
		);`,
	},
	"postgres": []string{
		`CREATE TABLE IF NOT EXISTS "staging_security_groups_spaces" (
			space_guid varchar(36) NOT NULL,
			security_group_guid varchar(36) NOT NULL,
			FOREIGN KEY (security_group_guid) REFERENCES security_groups(guid),
			PRIMARY KEY (space_guid, security_group_guid)
		);`,
	},
}

var migration_v0067c = map[string][]string{
	"mysql": []string{
		`CREATE TABLE IF NOT EXISTS "running_security_groups_spaces" (
			space_guid varchar(36) NOT NULL,
			security_group_guid varchar(36) NOT NULL,
			CONSTRAINT running_security_groups_spaces_id_fk
				FOREIGN KEY (security_group_guid)
				REFERENCES security_groups(guid),
			PRIMARY KEY (space_guid, security_group_guid)
		);`,
	},
	"postgres": []string{
		`CREATE TABLE IF NOT EXISTS "running_security_groups_spaces" (
			space_guid varchar(36) NOT NULL,
			security_group_guid varchar(36) NOT NULL,
			FOREIGN KEY (security_group_guid) REFERENCES security_groups(guid),
			PRIMARY KEY (space_guid, security_group_guid)
		);`,
	},
}
