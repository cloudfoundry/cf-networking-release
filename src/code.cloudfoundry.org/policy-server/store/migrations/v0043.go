package migrations

var migration_v0043 = map[string][]string{
	"mysql": {
		`ALTER TABLE terminals
		 MODIFY id INT,
		 DROP PRIMARY KEY,
		 ADD PRIMARY KEY (guid);`,
	},
	"postgres": {
		`ALTER TABLE terminals
		 DROP id,
		 ADD PRIMARY KEY (guid);`,
	},
}
