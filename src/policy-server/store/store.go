package store

import (
	"database/sql"
	"errors"
	"fmt"
	"policy-server/models"

	"github.com/jmoiron/sqlx"
)

var schemas = map[string][]string{
	"mysql": []string{
		`CREATE TABLE IF NOT EXISTS groups (
		id int NOT NULL AUTO_INCREMENT,
		guid varchar(255),
		PRIMARY KEY (id)
	);`,
		`CREATE TABLE IF NOT EXISTS destinations (
		id int NOT NULL AUTO_INCREMENT,
		group_id int REFERENCES groups(id),
		port int,
		protocol varchar(255),
		UNIQUE (group_id, port, protocol),
		PRIMARY KEY (id)
	);`,
		`CREATE TABLE IF NOT EXISTS policies (
		id int NOT NULL AUTO_INCREMENT,
		group_id int REFERENCES groups(id),
		destination_id int REFERENCES destinations(id),
		UNIQUE (group_id, destination_id)
	);`,
	},
	"postgres": []string{
		`CREATE TABLE IF NOT EXISTS groups (
		id SERIAL PRIMARY KEY,
		guid text
	);`,
		`CREATE TABLE IF NOT EXISTS destinations (
		id SERIAL PRIMARY KEY,
		group_id int REFERENCES groups(id),
		port int,
		protocol text,
		UNIQUE (group_id, port, protocol)
	);`,
		`CREATE TABLE IF NOT EXISTS policies (
		id SERIAL PRIMARY KEY,
		group_id int REFERENCES groups(id),
		destination_id int REFERENCES destinations(id),
		UNIQUE (group_id, destination_id)
	);`,
	},
}

//go:generate counterfeiter -o ../fakes/store.go --fake-name Store . Store
type Store interface {
	Create([]models.Policy) error
	All() ([]models.Policy, error)
	Delete([]models.Policy) error
}

//go:generate counterfeiter -o ../fakes/db.go --fake-name Db . db
type db interface {
	Beginx() (*sqlx.Tx, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	NamedExec(query string, arg interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
	DriverName() string
}

type Transaction interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Commit() error
	Rollback() error
	Rebind(string) string
}

var RecordNotFoundError = errors.New("record not found")

type store struct {
	conn        db
	group       GroupRepo
	destination DestinationRepo
	policy      PolicyRepo
}

func New(dbConnectionPool db, g GroupRepo, d DestinationRepo, p PolicyRepo) (Store, error) {
	err := setupTables(dbConnectionPool)
	if err != nil {
		return nil, fmt.Errorf("setting up tables: %s", err)
	}

	return &store{
		conn:        dbConnectionPool,
		group:       g,
		destination: d,
		policy:      p,
	}, nil
}

func rollback(tx Transaction, err error) error {
	txErr := tx.Rollback()
	if txErr != nil {
		return fmt.Errorf("db rollback: %s (sql error: %s)", txErr, err)
	}

	return err
}

func (s *store) Create(policies []models.Policy) error {
	tx, err := s.conn.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %s", err)
	}

	for _, policy := range policies {
		source_group_id, err := s.group.Create(tx, policy.Source.ID)
		if err != nil {
			return rollback(tx, fmt.Errorf("creating group: %s", err))
		}

		destination_group_id, err := s.group.Create(tx, policy.Destination.ID)
		if err != nil {
			return rollback(tx, fmt.Errorf("creating group: %s", err))
		}

		destination_id, err := s.destination.Create(tx, destination_group_id, policy.Destination.Port, policy.Destination.Protocol)
		if err != nil {
			return rollback(tx, fmt.Errorf("creating destination: %s", err))
		}

		err = s.policy.Create(tx, source_group_id, destination_id)
		if err != nil {
			return rollback(tx, fmt.Errorf("creating policy: %s", err))
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %s", err) // untested
	}

	return nil
}

func (s *store) Delete(policies []models.Policy) error {
	tx, err := s.conn.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %s", err)
	}

	for _, p := range policies {
		sourceGroupID, err := s.group.GetID(tx, p.Source.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			} else {
				return rollback(tx, fmt.Errorf("getting source id: %s", err))
			}
		}

		destGroupID, err := s.group.GetID(tx, p.Destination.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			} else {
				return rollback(tx, fmt.Errorf("getting destination group id: %s", err))
			}
		}

		destID, err := s.destination.GetID(tx, destGroupID, p.Destination.Port, p.Destination.Protocol)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			} else {
				return rollback(tx, fmt.Errorf("getting destination id: %s", err))
			}
		}

		err = s.policy.Delete(tx, sourceGroupID, destID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			} else {
				return rollback(tx, fmt.Errorf("deleting policy: %s", err))
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %s", err) // untested
	}

	return nil
}

func (s *store) All() ([]models.Policy, error) {
	policies := []models.Policy{}

	rows, err := s.conn.Query(`
		select
			src_grp.guid,
			dst_grp.guid,
			destinations.port,
			destinations.protocol
		from policies
		left outer join groups as src_grp on (policies.group_id = src_grp.id)
		left outer join destinations on (destinations.id = policies.destination_id)
		left outer join groups as dst_grp on (destinations.group_id = dst_grp.id);`)
	if err != nil {
		return nil, fmt.Errorf("listing all: %s", err)
	}

	for rows.Next() {
		var source_id, destination_id, protocol string
		var port int
		err = rows.Scan(&source_id, &destination_id, &port, &protocol)
		if err != nil {
			return nil, fmt.Errorf("listing all: %s", err)
		}

		policies = append(policies, models.Policy{
			Source: models.Source{
				ID: source_id,
			},
			Destination: models.Destination{
				ID:       destination_id,
				Protocol: protocol,
				Port:     port,
			},
		})
	}

	return policies, nil
}

func setupTables(dbConnectionPool db) error {
	driverName := dbConnectionPool.DriverName()
	schema, ok := schemas[driverName]
	if !ok {
		panic("unsupported DB DriverName")
	}
	for _, table := range schema {
		_, err := dbConnectionPool.Exec(table)
		if err != nil {
			return err
		}
	}
	return nil
}
