package store

import (
	"database/sql"
	"fmt"
	"policy-server/store/helpers"
	"strings"

	"policy-server/db"
	"policy-server/store/migrations"

	"github.com/jmoiron/sqlx"
)

//go:generate counterfeiter -o fakes/migrator.go --fake-name Migrator . Migrator
type Migrator interface {
	PerformMigrations(driverName string, migrationDb migrations.MigrationDb, maxNumMigrations int) (int, error)
}

//go:generate counterfeiter -o fakes/store.go --fake-name Store . Store
type Store interface {
	Create([]Policy) error
	All() ([]Policy, error)
	Delete([]Policy) error
	ByGuids([]string, []string, bool) ([]Policy, error)
	CheckDatabase() error
}

//go:generate counterfeiter -o fakes/database.go --fake-name Db . Database
type Database interface {
	Beginx() (db.Transaction, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	NamedExec(query string, arg interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
	DriverName() string
	RawConnection() *sqlx.DB
	Rebind(string) string
}

type store struct {
	conn        Database
	group       GroupRepo
	destination DestinationRepo
	policy      PolicyRepo
	tagLength   int
}

func New(dbConnectionPool Database, g GroupRepo, d DestinationRepo, p PolicyRepo, tl int) Store {
	return &store{
		conn:        dbConnectionPool,
		group:       g,
		destination: d,
		policy:      p,
		tagLength:   tl,
	}
}

func (s *store) Create(policies []Policy) error {
	tx, err := s.conn.Beginx()
	if err != nil {
		return fmt.Errorf("create transaction: %s", err)
	}

	err = s.createWithTx(tx, policies)
	if err != nil {
		return rollback(tx, err)
	}

	return commit(tx)
}

func (s *store) Delete(policies []Policy) error {
	tx, err := s.conn.Beginx()
	if err != nil {
		return fmt.Errorf("create transaction: %s", err)
	}

	err = s.deleteWithTx(tx, policies)
	if err != nil {
		return rollback(tx, err)
	}

	return commit(tx)
}

func (s *store) CheckDatabase() error {
	var result int
	return s.conn.QueryRow("SELECT 1").Scan(&result)
}

func (s *store) createWithTx(tx db.Transaction, policies []Policy) error {
	for _, policy := range policies {
		sourceGroupId, err := s.group.Create(tx, policy.Source.ID, "app")
		if err != nil {
			return fmt.Errorf("creating group: %s", err)
		}

		destinationGroupId, err := s.group.Create(tx, policy.Destination.ID, "app")
		if err != nil {
			return fmt.Errorf("creating group: %s", err)
		}

		destinationId, err := s.destination.Create(
			tx,
			destinationGroupId,
			policy.Destination.Port,
			policy.Destination.Ports.Start,
			policy.Destination.Ports.End,
			policy.Destination.Protocol,
		)
		if err != nil {
			return fmt.Errorf("creating destination: %s", err)
		}

		err = s.policy.Create(tx, sourceGroupId, destinationId)
		if err != nil {
			return fmt.Errorf("creating policy: %s", err)
		}
	}
	return nil
}

func (s *store) deleteWithTx(tx db.Transaction, policies []Policy) error {
	for _, p := range policies {
		sourceGroupID, err := s.group.GetID(tx, p.Source.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			} else {
				return fmt.Errorf("getting source id: %s", err)
			}
		}

		destGroupID, err := s.group.GetID(tx, p.Destination.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			} else {
				return fmt.Errorf("getting destination group id: %s", err)
			}
		}

		destID, err := s.destination.GetID(
			tx,
			destGroupID,
			p.Destination.Port,
			p.Destination.Ports.Start,
			p.Destination.Ports.End,
			p.Destination.Protocol,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			} else {
				return fmt.Errorf("getting destination id: %s", err)
			}
		}

		err = s.policy.Delete(tx, sourceGroupID, destID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			} else {
				return fmt.Errorf("deleting policy: %s", err)
			}
		}

		destIDCount, err := s.policy.CountWhereDestinationID(tx, destID)
		if err != nil {
			return fmt.Errorf("counting destination id: %s", err)
		}
		if destIDCount == 0 {
			err = s.destination.Delete(tx, destID)
			if err != nil {
				return fmt.Errorf("deleting destination: %s", err)
			}
		}

		err = s.deleteGroupRowIfLast(tx, sourceGroupID)
		if err != nil {
			return fmt.Errorf("deleting group row: %s", err)
		}

		err = s.deleteGroupRowIfLast(tx, destGroupID)
		if err != nil {
			return fmt.Errorf("deleting group row: %s", err)
		}
	}
	return nil
}

func (s *store) deleteGroupRowIfLast(tx db.Transaction, groupId int) error {
	policiesGroupIDCount, err := s.policy.CountWhereGroupID(tx, groupId)
	if err != nil {
		return err
	}

	destinationsGroupIDCount, err := s.destination.CountWhereGroupID(tx, groupId)
	if err != nil {
		return err
	}

	if policiesGroupIDCount == 0 && destinationsGroupIDCount == 0 {
		err = s.group.Delete(tx, groupId)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *store) policiesQuery(query string, args ...interface{}) ([]Policy, error) {
	var policies []Policy
	rebindedQuery := helpers.RebindForSQLDialect(query, s.conn.DriverName())

	rows, err := s.conn.Query(rebindedQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("listing all: %s", err)
	}

	defer rows.Close() // untested
	for rows.Next() {
		var sourceId, destinationId, protocol string
		var port, startPort, endPort, sourceTag, destinationTag int
		err = rows.Scan(
			&sourceId,
			&sourceTag,
			&destinationId,
			&destinationTag,
			&port,
			&startPort,
			&endPort,
			&protocol,
		)
		if err != nil {
			return nil, fmt.Errorf("listing all: %s", err)
		}

		policies = append(policies, Policy{
			Source: Source{
				ID:  sourceId,
				Tag: s.tagIntToString(sourceTag),
			},
			Destination: Destination{
				ID:       destinationId,
				Tag:      s.tagIntToString(destinationTag),
				Protocol: protocol,
				Port:     port,
				Ports: Ports{
					Start: startPort,
					End:   endPort,
				},
			},
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("listing all, getting next row: %s", err) // untested
	}
	return policies, nil
}

func (s *store) ByGuids(srcGuids, destGuids []string, inSourceAndDest bool) ([]Policy, error) {
	numSourceGuids := len(srcGuids)
	numDestinationGuids := len(destGuids)
	if numSourceGuids == 0 && numDestinationGuids == 0 {
		return []Policy{}, nil
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
			destinations.start_port,
			destinations.end_port,
			destinations.protocol
		from policies
		left outer join groups as src_grp on (policies.group_id = src_grp.id)
		left outer join destinations on (destinations.id = policies.destination_id)
		left outer join groups as dst_grp on (destinations.group_id = dst_grp.id)`

	if len(wheres) > 0 {
		andOr := " OR "
		if inSourceAndDest {
			andOr = " AND "
		}
		query += " where " + strings.Join(wheres, andOr)
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

func (s *store) All() ([]Policy, error) {
	return s.policiesQuery(`
		select
			src_grp.guid,
			src_grp.id,
			dst_grp.guid,
			dst_grp.id,
			destinations.port,
			destinations.start_port,
			destinations.end_port,
			destinations.protocol
		from policies
		left outer join groups as src_grp on (policies.group_id = src_grp.id)
		left outer join destinations on (destinations.id = policies.destination_id)
		left outer join groups as dst_grp on (destinations.group_id = dst_grp.id);`)
}

func (s *store) tagIntToString(tag int) string {
	return fmt.Sprintf("%"+fmt.Sprintf("0%d", s.tagLength*2)+"X", tag)
}
