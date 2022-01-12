package migrations

var migration_v0067 = map[string][]string{
	"mysql": []string{
		`CREATE TABLE IF NOT EXISTS "security_groups" (
			id integer NOT NULL AUTO_INCREMENT PRIMARY KEY,
			guid varchar(36) NOT NULL,
			name varchar(255) NOT NULL,
			rules mediumtext,
			staging_default bool DEFAULT false,
			running_default bool DEFAULT false,
			staging_spaces json,
			running_spaces json,
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
			staging_spaces jsonb,
			running_spaces jsonb,
			UNIQUE (guid)
		);`,
	},
}
