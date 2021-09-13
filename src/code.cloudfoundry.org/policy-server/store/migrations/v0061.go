package migrations

var migration_v0061 = map[string][]string{
	"mysql": {
		`CREATE PROCEDURE drop_terminal_guid_unique_index()
		BEGIN
			SELECT
				index_name INTO @name
			FROM information_schema.statistics
			WHERE
				table_name='ip_ranges'
				AND column_name='terminal_guid'
				AND non_unique=0;

			SET @query = CONCAT('ALTER TABLE ip_ranges DROP INDEX ', @name);

			PREPARE stmt FROM @query;

			EXECUTE stmt;

 			DEALLOCATE PREPARE stmt;
			SET @name = NULL;
		END;`,
	},
	"postgres": {
		`ALTER TABLE ip_ranges DROP CONSTRAINT ip_ranges_terminal_guid_unique`,
	},
}
