package migrations

var migration_v0001 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS ` + "`groups`" + ` (
		id int NOT NULL AUTO_INCREMENT,
		guid varchar(255),
		UNIQUE (guid),
		PRIMARY KEY (id)
	);`,
		`CREATE TABLE IF NOT EXISTS destinations (
		id int NOT NULL AUTO_INCREMENT,
		group_id int REFERENCES ` + "`groups`" + `(id),
		port int,
		protocol varchar(255),
		UNIQUE (group_id, port, protocol),
		PRIMARY KEY (id)
	);`,
		`CREATE TABLE IF NOT EXISTS policies (
		id int NOT NULL AUTO_INCREMENT,
		group_id int REFERENCES ` + "`groups`" + `(id),
		destination_id int REFERENCES destinations(id),
		UNIQUE (group_id, destination_id),
		PRIMARY KEY (id)
	);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS groups (
		id SERIAL PRIMARY KEY,
		guid text,
		UNIQUE (guid)
	);`,
		`CREATE TABLE IF NOT EXISTS destinations (
		id SERIAL PRIMARY KEY,
		group_id int REFERENCES groups(id),
		port int,
		protocol text,
		UNIQUE (group_id, port, protocol)
	);`,
		`CREATE TABLE IF NOT EXISTS policies (
		id SERIAL PRIMARY KEY,
		group_id int REFERENCES groups(id),
		destination_id int REFERENCES destinations(id),
		UNIQUE (group_id, destination_id)
	);`,
	},
}

var migration_modified_v0001 = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS ` + "`groups`" + ` (
		id int NOT NULL AUTO_INCREMENT,
		guid varchar(255),
		UNIQUE (guid),
		PRIMARY KEY (id)
	);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS groups (
		id SERIAL PRIMARY KEY,
		guid text,
		UNIQUE (guid)
	);`,
	},
}

var migration_modified_v0001a = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS destinations (
		id int NOT NULL AUTO_INCREMENT,
		group_id int REFERENCES ` + "`groups`" + `(id),
		port int,
		protocol varchar(255),
		UNIQUE (group_id, port, protocol),
		PRIMARY KEY (id)
	);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS destinations (
		id SERIAL PRIMARY KEY,
		group_id int REFERENCES groups(id),
		port int,
		protocol text,
		UNIQUE (group_id, port, protocol)
	);`,
	},
}

var migration_modified_v0001b = map[string][]string{
	"mysql": {
		`CREATE TABLE IF NOT EXISTS policies (
		id int NOT NULL AUTO_INCREMENT,
		group_id int REFERENCES ` + "`groups`" + `(id),
		destination_id int REFERENCES destinations(id),
		UNIQUE (group_id, destination_id),
		PRIMARY KEY (id)
	);`,
	},
	"postgres": {
		`CREATE TABLE IF NOT EXISTS policies (
		id SERIAL PRIMARY KEY,
		group_id int REFERENCES groups(id),
		destination_id int REFERENCES destinations(id),
		UNIQUE (group_id, destination_id)
	);`,
	},
}
