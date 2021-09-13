package migrations

var migration_v0002 = map[string][]string{
	"mysql": {
		`ALTER TABLE destinations ADD COLUMN start_port int;`,
		`ALTER TABLE destinations ADD COLUMN end_port int;`,
		`UPDATE destinations SET start_port = port;`,
		`UPDATE destinations SET end_port = port;`,
		`CREATE PROCEDURE drop_destination_index()
BEGIN
 SELECT DATABASE() FROM DUAL INTO @databaseName;
 SELECT CONSTRAINT_NAME INTO @name
 FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
 WHERE TABLE_NAME='destinations' AND COLUMN_NAME= 'port' AND TABLE_SCHEMA=@databaseName;

 SET @query = CONCAT('ALTER TABLE destinations DROP INDEX ', @name);

 PREPARE stmt FROM @query;

 EXECUTE stmt;

 DEALLOCATE PREPARE stmt;
 SET @databaseName = NULL;
 SET @query = NULL;
 SET @name = NULL;

END;`,
		`CALL drop_destination_index();`,
		`ALTER TABLE destinations ADD UNIQUE key unique_destination (group_id, start_port, end_port, protocol);`,
	},
	"postgres": {
		`ALTER TABLE destinations ADD COLUMN start_port int;`,
		`ALTER TABLE destinations ADD COLUMN end_port int;`,
		`UPDATE destinations SET start_port = port;`,
		`UPDATE destinations SET end_port = port;`,
		`DO $$DECLARE r record;
		 	BEGIN
		 		FOR r in select CONSTRAINT_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE WHERE TABLE_NAME='destinations' AND COLUMN_NAME='port'
		 		LOOP
		 			EXECUTE 'ALTER TABLE destinations DROP CONSTRAINT ' || quote_ident(r.CONSTRAINT_NAME);
		 		END LOOP;
		 	END$$;
	`,
		`ALTER TABLE destinations ADD CONSTRAINT unique_destination UNIQUE (group_id, start_port, end_port, protocol);`,
	},
}

var migration_modified_v0002 = map[string][]string{
	"mysql": {
		`ALTER TABLE destinations ADD COLUMN start_port int;`,
	},
	"postgres": {
		`ALTER TABLE destinations ADD COLUMN start_port int;`,
	},
}

var migration_modified_v0002a = map[string][]string{
	"mysql": {
		`ALTER TABLE destinations ADD COLUMN end_port int;`,
	},
	"postgres": {
		`ALTER TABLE destinations ADD COLUMN end_port int;`,
	},
}

var migration_modified_v0002b = map[string][]string{
	"mysql": {
		`UPDATE destinations SET start_port = port;`,
	},
	"postgres": {
		`UPDATE destinations SET start_port = port;`,
	},
}

var migration_modified_v0002c = map[string][]string{
	"mysql": {
		`UPDATE destinations SET end_port = port;`,
	},
	"postgres": {
		`UPDATE destinations SET end_port = port;`,
	},
}

var migration_modified_v0002d = map[string][]string{
	"mysql": {
		`CREATE PROCEDURE drop_destination_index()
BEGIN
 SELECT DATABASE() FROM DUAL INTO @databaseName;
 SELECT CONSTRAINT_NAME INTO @name
 FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
 WHERE TABLE_NAME='destinations' AND COLUMN_NAME= 'port' AND TABLE_SCHEMA=@databaseName;

 SET @query = CONCAT('ALTER TABLE destinations DROP INDEX ', @name);

 PREPARE stmt FROM @query;

 EXECUTE stmt;

 DEALLOCATE PREPARE stmt;
 SET @databaseName = NULL;
 SET @query = NULL;
 SET @name = NULL;

END;`,
	},
	"postgres": {
		`DO $$DECLARE r record;
		 	BEGIN
		 		FOR r in select CONSTRAINT_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE WHERE TABLE_NAME='destinations' AND COLUMN_NAME='port'
		 		LOOP
		 			EXECUTE 'ALTER TABLE destinations DROP CONSTRAINT ' || quote_ident(r.CONSTRAINT_NAME);
		 		END LOOP;
		 	END$$;
	`,
	},
}

var migration_modified_v0002e = map[string][]string{
	"mysql": {
		`CALL drop_destination_index();`,
	},
	"postgres": {},
}

var migration_modified_v0002f = map[string][]string{
	"mysql": {
		`ALTER TABLE destinations ADD UNIQUE key unique_destination (group_id, start_port, end_port, protocol);`,
	},
	"postgres": {
		`ALTER TABLE destinations ADD CONSTRAINT unique_destination UNIQUE (group_id, start_port, end_port, protocol);`,
	},
}
