package store

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type SecurityGroups []SecurityGroup

type SecurityGroup struct {
	Guid              string     `json:"guid"`
	Name              string     `json:"name"`
	Rules             string     `json:"rules"`
	StagingDefault    bool       `json:"staging_default"`
	RunningDefault    bool       `json:"running_default"`
	StagingSpaceGuids SpaceGuids `json:"staging_space_guids"`
	RunningSpaceGuids SpaceGuids `json:"running_space_guids"`
}

func (sgs SecurityGroups) Value() (driver.Value, error) {
	return json.Marshal(sgs)
}

func (sgs *SecurityGroups) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	err := json.Unmarshal(b, &sgs)
	if err != nil {
		return err
	}
	return nil
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
