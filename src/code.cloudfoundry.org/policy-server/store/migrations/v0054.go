package migrations

var migration_v0054 = map[string][]string{
	"mysql": {
		`ALTER TABLE egress_policies
		 MODIFY id INT,
		 DROP PRIMARY KEY,
		 ADD PRIMARY KEY (guid);`,
	},
	"postgres": {
		`ALTER TABLE egress_policies
		 DROP id,
		 ADD PRIMARY KEY (guid);`,
	},
}
