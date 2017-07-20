package migrations

var migration_v0002 = map[string][]string{
	"mysql": {
		`ALTER TABLE destinations ADD COLUMN start_port int;`,
		`ALTER TABLE destinations ADD COLUMN end_port int;`,
		`UPDATE destinations SET start_port = port;`,
		`UPDATE destinations SET end_port = port;`,
		`CREATE PROCEDURE drop_destination_index()
BEGIN
 SELECT CONSTRAINT_NAME INTO @name
 FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE t1
 WHERE TABLE_NAME='destinations' AND COLUMN_NAME= 'port';

 SET @query = CONCAT('ALTER TABLE destinations DROP INDEX ', @name);

 PREPARE stmt FROM @query;

 EXECUTE stmt;

 DEALLOCATE PREPARE stmt;
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

