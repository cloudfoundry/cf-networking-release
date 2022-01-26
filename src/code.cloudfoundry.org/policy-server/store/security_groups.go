package store

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type SecurityGroup struct {
	Guid              string
	Name              string
	Rules             string
	StagingDefault    bool
	RunningDefault    bool
	StagingSpaceGuids SpaceGuids
	RunningSpaceGuids SpaceGuids
}

type SpaceGuids []string

func (guids SpaceGuids) Value() (driver.Value, error) {
	return json.Marshal(guids)
}

func (guids *SpaceGuids) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	err := json.Unmarshal(b, &guids)
	if err != nil {
		return err
	}

	return nil
}

type Page struct {
	Limit int
	From  int
}

type Pagination struct {
	Next int
}
