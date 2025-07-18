package migrations

var migration_v0097 = map[string][]string{
	"mysql": {
		`INSERT INTO running_security_groups_spaces (security_group_guid, space_guid) SELECT
			guid AS security_group_guid,
			JSON_UNQUOTE(JSON_EXTRACT(security_groups.running_spaces, CONCAT('$[', x.id, ']'))) AS space_guid     
		FROM
			security_groups
		JOIN 
			(SELECT id from temp_sequence) x
		WHERE x.id < JSON_LENGTH(security_groups.running_spaces)`,
	},
	"postgres": {
		`INSERT INTO running_security_groups_spaces (security_group_guid, space_guid) SELECT
			guid AS security_group_guid,
			jsonb_array_elements_text(running_spaces) AS space_id
		FROM security_groups`,
	},
}
