package migrations

var migration_v0095 = map[string][]string{
	"mysql": {`

CREATE PROCEDURE generate_sequence()
BEGIN
    DECLARE max_length INT;

    SELECT GREATEST(MAX(JSON_LENGTH(running_spaces)), MAX(JSON_LENGTH(staging_spaces))) INTO max_length
    FROM security_groups;

    CREATE TABLE IF NOT EXISTS temp_sequence (id INT NOT NULL PRIMARY KEY);

    SET @counter = 0;

    WHILE @counter < max_length DO
        INSERT INTO temp_sequence (id) VALUES (@counter);
        SET @counter = @counter + 1;
    END WHILE;
END
`},
	"postgres": {},
}
