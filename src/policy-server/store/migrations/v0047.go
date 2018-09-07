package migrations

var migration_v0047 = map[string][]string{
	"mysql": {
		`CREATE INDEX ip_ranges_terminal_guid_idx ON ip_ranges (terminal_guid);`,
	},
	"postgres": {
		`CREATE INDEX ip_ranges_terminal_guid_idx ON ip_ranges (terminal_guid);`,
	},
}
