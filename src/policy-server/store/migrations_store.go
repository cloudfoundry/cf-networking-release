package store

import (
	"database/sql"
	"fmt"
	"policy-server/db"
	"strings"
)

type MigrationsStore struct {
	DBConn *db.ConnWrapper
}

func (m *MigrationsStore) HasV1MigrationOccurred() (bool, error) {
	if !m.tableExists("gorp_migrations") {
		return false, nil
	}

	migrationIDExists, err := m.migrationIDExists("1")
	if err != nil {
		return false, fmt.Errorf("failed to query migration id: %s", err)
	}
	if !migrationIDExists {
		return false, nil
	}

	if !m.tableExists("destinations") || !m.tableExists("policies") {
		return false, nil
	}

	return true, nil
}

func (m *MigrationsStore) HasV2MigrationOccurred() (bool, error) {
	if !m.tableExists("gorp_migrations") {
		return false, nil
	}

	migrationIDExists, err := m.migrationIDExists("2")
	if err != nil {
		return false, fmt.Errorf("failed to query migration id: %s", err)
	}

	if !migrationIDExists {
		return false, nil
	}

	//This constraint is added as the last step of the V2 migration(s)
	query := `SELECT constraint_name FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
				WHERE constraint_name = 'unique_destination'
					AND table_name = 'destinations'
			    LIMIT 1`

	rows, err := m.DBConn.Query(query)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	if err != nil {
		return false, fmt.Errorf("failed to query constraint: %s", err)
	}

	return rows.Next(), nil
}

func (m *MigrationsStore) HasV3MigrationOccurred() (bool, error) {
	if !m.tableExists("gorp_migrations") {
		return false, nil
	}

	migrationIDExists, err := m.migrationIDExists("3")
	if err != nil {
		return false, fmt.Errorf("failed to query migration id: %s", err)
	}
	if !migrationIDExists {
		return false, nil
	}

	var query, index string

	if m.DBConn.DriverName() == "mysql" {
		query = `SELECT 1 FROM information_schema.statistics WHERE table_name = 'groups' AND index_name = 'idx_type'`
	} else {
		query = `SELECT 1 FROM pg_class c WHERE c.relname = 'idx_type'`
	}

	err = m.DBConn.QueryRow(query).Scan(&index)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to query column: %s", err)
	}

	return true, nil
}

func (m *MigrationsStore) tableExists(tableName string) bool {
	rows, err := m.DBConn.Query(fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", tableName))
	if err != nil {
		return false
	}
	rows.Close()
	return true
}

func (m *MigrationsStore) migrationIDExists(ids ...string) (bool, error) {
	var countIds int

	allIds := `'` + strings.Join(ids, `','`) + `'`

	query := fmt.Sprintf(`SELECT count(id) FROM gorp_migrations WHERE id IN (%s)`, allIds)
	err := m.DBConn.QueryRow(query).Scan(&countIds)
	if err != nil {
		return false, err
	}

	return countIds == len(ids), nil
}
