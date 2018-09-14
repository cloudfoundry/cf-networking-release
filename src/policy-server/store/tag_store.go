package store

import (
	"fmt"
	"policy-server/db"
)

//go:generate counterfeiter -o fakes/tag_store.go --fake-name TagStore . TagStore
type TagStore interface {
	CreateTag(string, string) (Tag, error)
	Tags() ([]Tag, error)
}

type tagStore struct {
	conn      Database
	group     GroupRepo
	tagLength int
}

func NewTagStore(dbConnectionPool Database, groupRepo GroupRepo, tagLength int) *tagStore {
	return &tagStore{
		conn:      dbConnectionPool,
		group:     groupRepo,
		tagLength: tagLength,
	}
}

func (s *tagStore) CreateTag(groupGuid, groupType string) (Tag, error) {
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

func (s *tagStore) Tags() ([]Tag, error) {
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

func (s *tagStore) tagIntToString(tag int) string {
	return fmt.Sprintf("%"+fmt.Sprintf("0%d", s.tagLength*2)+"X", tag)
}

func commit(tx db.Transaction) error {
	err := tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %s", err)
	}
	return nil
}

func rollback(tx db.Transaction, err error) error {
	txErr := tx.Rollback()
	if txErr != nil {
		return fmt.Errorf("database rollback: %s (sql error: %s)", txErr, err)
	}
	return err
}
