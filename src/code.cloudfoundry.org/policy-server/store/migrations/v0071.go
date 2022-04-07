package migrations

// Dropping Dynamic Egress tables
// There are 7 tables related to dynamic egress policies:
// apps, defaults, destination_metadatas, egress_policies, ip_ranges, terminals, and spaces.

var migration_v0071 = map[string][]string{
	"mysql": []string{
		`DROP TABLE IF EXISTS spaces;`,
	},
	"postgres": []string{
		`DROP TABLE IF EXISTS spaces;`,
	},
}
