package migrations

var migration_v0087 = map[string][]string{
	"mysql": {
		`CREATE PROCEDURE drop_staging_spaces_index()
			BEGIN
				SET @exist := (select count(*) from information_schema.statistics where table_name = 'security_groups' and index_name = 'staging_spaces_idx' and table_schema = database());
				SET @sqlstmt := if( @exist <=> 0, 'select ''INFO: Index does not exist''', 'alter table security_groups drop index staging_spaces_idx');
				PREPARE stmt FROM @sqlstmt;
				EXECUTE stmt;

				DEALLOCATE PREPARE stmt;
			END;`,
	},
	"postgres": {},
}
