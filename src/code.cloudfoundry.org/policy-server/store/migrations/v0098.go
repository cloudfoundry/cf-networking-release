package migrations

var migration_v0098 = map[string][]string{
	"mysql": {
		`INSERT INTO staging_security_groups_spaces (security_group_guid, space_guid) SELECT
			guid AS security_group_guid,
			JSON_UNQUOTE(JSON_EXTRACT(security_groups.staging_spaces, CONCAT('$[', x.id, ']'))) AS space_guid     
		FROM
			security_groups
		JOIN 
			(SELECT id from temp_sequence) x
		WHERE x.id < JSON_LENGTH(security_groups.staging_spaces) ON DUPLICATE KEY UPDATE space_guid = VALUES(space_guid), security_group_guid = VALUES(security_group_guid)`,
	},
	"postgres": {
		`INSERT INTO staging_security_groups_spaces (security_group_guid, space_guid) SELECT
			guid AS security_group_guid,
			jsonb_array_elements_text(staging_spaces) AS space_id
		FROM security_groups ON CONFLICT (security_group_guid, space_guid) DO NOTHING`,
	},
}
