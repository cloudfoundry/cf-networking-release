package store

import uuid "github.com/nu7hatch/gouuid"

//go:generate counterfeiter -o fakes/guid_generator.go --fake-name GUIDGenerator . guidGenerator
type guidGenerator interface {
	New() string
}

type GuidGenerator struct{}

func (g *GuidGenerator) New() string {
	guid, err := uuid.NewV4()
	if err != nil {
		// this only happens if the system can't make random numbers
		// we can't recover from this, so just crash
		panic(err)
	}
	return guid.String()
}
