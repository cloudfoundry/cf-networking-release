package adapter

import uuid "github.com/nu7hatch/gouuid"

type UUIDAdapter struct{}

func (*UUIDAdapter) GenerateUUID() (string, error) {
	uuid, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return string(uuid.String()), err
}
