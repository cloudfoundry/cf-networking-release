package store

var Schemas = map[string][]string{
	"mysql": []string{
		`CREATE TABLE IF NOT EXISTS groups (
		id int NOT NULL AUTO_INCREMENT,
		guid varchar(255),
		UNIQUE (guid),
		PRIMARY KEY (id)
	);`,
		`CREATE TABLE IF NOT EXISTS destinations (
		id int NOT NULL AUTO_INCREMENT,
		group_id int REFERENCES groups(id),
		port int,
		protocol varchar(255),
		UNIQUE (group_id, port, protocol),
		PRIMARY KEY (id)
	);`,
		`CREATE TABLE IF NOT EXISTS policies (
		id int NOT NULL AUTO_INCREMENT,
		group_id int REFERENCES groups(id),
		destination_id int REFERENCES destinations(id),
		UNIQUE (group_id, destination_id),
		PRIMARY KEY (id)
	);`,
	},
	"postgres": []string{
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

var SchemasV1Up = map[string][]string{
	"mysql": []string{
		`ALTER TABLE destinations ADD COLUMN start_port int;`,
		`ALTER TABLE destinations ADD COLUMN end_port int;`,
		`UPDATE destinations SET start_port = port;`,
		`UPDATE destinations SET end_port = port;`,
	},
	"postgres": []string{
		`ALTER TABLE destinations ADD COLUMN start_port int;`,
		`ALTER TABLE destinations ADD COLUMN end_port int;`,
		`UPDATE destinations SET start_port = port;`,
		`UPDATE destinations SET end_port = port;`,
	},
}

var SchemasV1Down = map[string][]string{
	"mysql": []string{
		`ALTER TABLE destinations DROP COLUMN start_port;`,
		`ALTER TABLE destinations DROP COLUMN end_port;`,
	},
	"postgres": []string{
		`ALTER TABLE destinations DROP COLUMN start_port;`,
		`ALTER TABLE destinations DROP COLUMN end_port;`,
	},
}
