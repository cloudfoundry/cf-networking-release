package store

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"policy-server/models"
	"policy-server/store/helpers"
	"strings"

	"github.com/jmoiron/sqlx"
)

var schemas = map[string][]string{
	"mysql": []string{
		`CREATE TABLE IF NOT EXISTS groups (
		id int NOT NULL AUTO_INCREMENT,
		guid varchar(255),
		UNIQUE (guid),
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
		UNIQUE (group_id, destination_id),
		PRIMARY KEY (id)
	);`,
	},
	"postgres": []string{
		`CREATE TABLE IF NOT EXISTS groups (
		id SERIAL PRIMARY KEY,
		guid text,
		UNIQUE (guid)
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

//go:generate counterfeiter -o fakes/db.go --fake-name Db . db
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
	tagLength   int
}

const MAX_TAG_LENGTH = 3
const MIN_TAG_LENGTH = 1

func New(dbConnectionPool db, g GroupRepo, d DestinationRepo, p PolicyRepo, tl int) (Store, error) {
	if tl < MIN_TAG_LENGTH || tl > MAX_TAG_LENGTH {
		return nil, fmt.Errorf("tag length out of range (%d-%d): %d",
			MIN_TAG_LENGTH,
			MAX_TAG_LENGTH,
			tl,
		)
	}

	err := setupTables(dbConnectionPool)
	if err != nil {
		return nil, fmt.Errorf("setting up tables: %s", err)
	}

	err = populateTables(dbConnectionPool, tl)
	if err != nil {
		return nil, fmt.Errorf("populating tables: %s", err)
	}

	return &store{
		conn:        dbConnectionPool,
		group:       g,
		destination: d,
		policy:      p,
		tagLength:   tl,
	}, nil
}

func rollback(tx Transaction, err error) error {
	txErr := tx.Rollback()
	if txErr != nil {
		return fmt.Errorf("db rollback: %s (sql error: %s)", txErr, err)
	}

	return err
}

func (s *store) Create(ctx context.Context, policies []models.Policy) error {
	errorChan := make(chan error, 1)
	tx, err := s.conn.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %s", err)
	}
	go func() {
		for _, policy := range policies {
			source_group_id, err := s.group.Create(tx, policy.Source.ID)
			if err != nil {
				errorChan <- rollback(tx, fmt.Errorf("creating group: %s", err))
			}

			destination_group_id, err := s.group.Create(tx, policy.Destination.ID)
			if err != nil {
				errorChan <- rollback(tx, fmt.Errorf("creating group: %s", err))
			}

			destination_id, err := s.destination.Create(tx, destination_group_id, policy.Destination.Port, policy.Destination.Protocol)
			if err != nil {
				errorChan <- rollback(tx, fmt.Errorf("creating destination: %s", err))
			}

			err = s.policy.Create(tx, source_group_id, destination_id)
			if err != nil {
				errorChan <- rollback(tx, fmt.Errorf("creating policy: %s", err))
			}
		}

		err = tx.Commit()
		if err != nil {
			errorChan <- fmt.Errorf("commit transaction: %s", err) // TODO untested
		}
		errorChan <- nil
	}()
	for {
		select {
		case err := <-errorChan:
			return err
		case <-ctx.Done():
			return fmt.Errorf("context done")
		}
	}
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

		destIDCount, err := s.policy.CountWhereDestinationID(tx, destID)
		if err != nil {
			return rollback(tx, fmt.Errorf("counting destination id: %s", err))
		}
		if destIDCount == 0 {
			err = s.destination.Delete(tx, destID)
			if err != nil {
				return rollback(tx, fmt.Errorf("deleting destination: %s", err))
			}
		}

		err = s.deleteGroupRowIfLast(tx, sourceGroupID)
		if err != nil {
			return rollback(tx, fmt.Errorf("deleting group row: %s", err))
		}

		err = s.deleteGroupRowIfLast(tx, destGroupID)
		if err != nil {
			return rollback(tx, fmt.Errorf("deleting group row: %s", err))
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %s", err) // untested
	}

	return nil
}

func (s *store) deleteGroupRowIfLast(tx Transaction, group_id int) error {
	policiesGroupIDCount, err := s.policy.CountWhereGroupID(tx, group_id)
	if err != nil {
		return err
	}

	destinationsGroupIDCount, err := s.destination.CountWhereGroupID(tx, group_id)
	if err != nil {
		return err
	}

	if policiesGroupIDCount == 0 && destinationsGroupIDCount == 0 {
		err = s.group.Delete(tx, group_id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *store) policiesQuery(query string, args ...interface{}) ([]models.Policy, error) {
	policies := []models.Policy{}
	rebindedQuery := helpers.RebindForSQLDialect(query, s.conn.DriverName())

	rows, err := s.conn.Query(rebindedQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("listing all: %s", err)
	}

	for rows.Next() {
		var source_id, destination_id, protocol string
		var port, source_tag, destination_tag int
		err = rows.Scan(&source_id, &source_tag, &destination_id, &destination_tag, &port, &protocol)
		if err != nil {
			return nil, fmt.Errorf("listing all: %s", err)
		}

		policies = append(policies, models.Policy{
			Source: models.Source{
				ID:  source_id,
				Tag: s.tagIntToString(source_tag),
			},
			Destination: models.Destination{
				ID:       destination_id,
				Tag:      s.tagIntToString(destination_tag),
				Protocol: protocol,
				Port:     port,
			},
		})
	}
	return policies, nil
}

func (s *store) ByGuids(srcGuids, destGuids []string) ([]models.Policy, error) {
	numSourceGuids := len(srcGuids)
	numDestinationGuids := len(destGuids)
	if numSourceGuids == 0 && numDestinationGuids == 0 {
		return []models.Policy{}, nil
	}

	var wheres []string
	if numSourceGuids > 0 {
		wheres = append(wheres, fmt.Sprintf("src_grp.guid in (%s)", helpers.QuestionMarks(numSourceGuids)))
	}

	if numDestinationGuids > 0 {
		wheres = append(wheres, fmt.Sprintf("dst_grp.guid in (%s)", helpers.QuestionMarks(numDestinationGuids)))
	}

	query := `
		select
			src_grp.guid,
			src_grp.id,
			dst_grp.guid,
			dst_grp.id,
			destinations.port,
			destinations.protocol
		from policies
		left outer join groups as src_grp on (policies.group_id = src_grp.id)
		left outer join destinations on (destinations.id = policies.destination_id)
		left outer join groups as dst_grp on (destinations.group_id = dst_grp.id)`

	if len(wheres) > 0 {
		query += " where " + strings.Join(wheres, " OR ")
	}
	query += ";"

	whereBindings := make([]interface{}, numSourceGuids+numDestinationGuids)
	for i := 0; i < len(whereBindings); i++ {
		if i < numSourceGuids {
			whereBindings[i] = srcGuids[i]
		} else {
			whereBindings[i] = destGuids[i-numSourceGuids]
		}
	}

	return s.policiesQuery(query, whereBindings...)
}

func (s *store) All() ([]models.Policy, error) {
	return s.policiesQuery(`
		select
			src_grp.guid,
			src_grp.id,
			dst_grp.guid,
			dst_grp.id,
			destinations.port,
			destinations.protocol
		from policies
		left outer join groups as src_grp on (policies.group_id = src_grp.id)
		left outer join destinations on (destinations.id = policies.destination_id)
		left outer join groups as dst_grp on (destinations.group_id = dst_grp.id);`)
}

func (s *store) Tags() ([]models.Tag, error) {
	tags := []models.Tag{}

	rows, err := s.conn.Query(`
		SELECT guid, id FROM groups
		WHERE guid IS NOT NULL
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %s", err)
	}

	for rows.Next() {
		var id string
		var tag int

		err = rows.Scan(&id, &tag)
		if err != nil {
			return nil, fmt.Errorf("listing tags: %s", err)
		}

		tags = append(tags, models.Tag{
			ID:  id,
			Tag: s.tagIntToString(tag),
		})
	}

	return tags, nil
}

func (s *store) tagIntToString(tag int) string {
	return fmt.Sprintf("%"+fmt.Sprintf("0%d", s.tagLength*2)+"X", tag)
}

func setupTables(dbConnectionPool db) error {
	driverName := dbConnectionPool.DriverName()
	schema, ok := schemas[driverName]
	if !ok {
		return errors.New("unsupported DB DriverName")
	}

	for _, table := range schema {
		_, err := dbConnectionPool.Exec(table)
		if err != nil {
			return err
		}
	}
	return nil
}

func populateTables(dbConnectionPool db, tl int) error {
	var err error
	row := dbConnectionPool.QueryRow(`SELECT COUNT(*) FROM groups`)
	if row != nil {
		var count int
		err = row.Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	}

	var b bytes.Buffer
	_, err = b.WriteString("INSERT INTO groups (guid) VALUES (NULL)")
	if err != nil {
		return err
	}

	for i := 1; i < int(math.Exp2(float64(tl*8)))-1; i++ {
		_, err = b.WriteString(", (NULL)")
		if err != nil {
			return err
		}
	}

	_, err = dbConnectionPool.Exec(b.String())
	if err != nil {
		return err
	}

	return nil
}
