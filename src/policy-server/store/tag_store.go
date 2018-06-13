package store

import (
	"fmt"
)

//go:generate counterfeiter -o fakes/tag_store.go --fake-name TagStore . TagStore
type TagStore interface {
	CreateTag(string, string) (Tag, error)
	Tags() ([]Tag, error)
}

func NewTagStore(dbConnectionPool database, migrationDbConnectionPool database, g GroupRepo, tl int, migrator Migrator) (TagStore, error) {
	if tl < MinTagLength || tl > MaxTagLength {
		return nil, fmt.Errorf("tag length out of range (%d-%d): %d",
			MinTagLength,
			MaxTagLength,
			tl,
		)
	}

	_, err := migrator.PerformMigrations(migrationDbConnectionPool.DriverName(), migrationDbConnectionPool, 0)
	if err != nil {
		return nil, fmt.Errorf("perform migrations: %s", err)
	}

	err = populateTables(dbConnectionPool, tl)
	if err != nil {
		return nil, fmt.Errorf("populating tables: %s", err)
	}

	return &store{
		conn:      dbConnectionPool,
		group:     g,
		tagLength: tl,
	}, nil
}

func (s *store) CreateTag(groupGuid, groupType string) (Tag, error) {
	tx, err := s.conn.Beginx()
	if err != nil {
		return Tag{}, fmt.Errorf("begin transaction: %s", err)
	}

	tagID, err := s.group.Create(tx, groupGuid, groupType)
	if err != nil {
		return Tag{}, rollback(tx, err)
	}

	err = commit(tx)
	if err != nil {
		return Tag{}, rollback(tx, err)
	}

	return Tag{
		ID:   groupGuid,
		Tag:  s.tagIntToString(tagID),
		Type: groupType,
	}, nil
}

func (s *store) Tags() ([]Tag, error) {
	var tags []Tag

	rows, err := s.conn.Query(`
		SELECT guid, id, type FROM groups
		WHERE guid IS NOT NULL
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %s", err)
	}

	defer rows.Close() // untested
	for rows.Next() {
		var id string
		var tag int
		var groupType string

		err = rows.Scan(&id, &tag, &groupType)
		if err != nil {
			return nil, fmt.Errorf("listing tags: %s", err)
		}

		tags = append(tags, Tag{
			ID:   id,
			Tag:  s.tagIntToString(tag),
			Type: groupType,
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("listing tags, getting next row: %s", err) // untested
	}

	return tags, nil
}
