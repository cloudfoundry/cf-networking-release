package port_allocator

import (
	"fmt"
	"io"
	"os"
)

//go:generate counterfeiter -o ../fakes/tracker.go --fake-name Tracker . tracker
type tracker interface {
	AcquireOne(pool *Pool) (int, error)
	ReleaseMany(pool *Pool, toRelease []int) error
}

//go:generate counterfeiter -o ../fakes/serializer.go --fake-name Serializer . serializer
type serializer interface {
	DecodeAll(file io.ReadSeeker, outData interface{}) error
	EncodeAndOverwrite(file OverwriteableFile, outData interface{}) error
}

//go:generate counterfeiter -o ../fakes/file_locker.go --fake-name FileLocker . fileLocker
type fileLocker interface {
	Open() (*os.File, error)
}

type PortAllocator struct {
	Tracker    tracker
	Serializer serializer
	Locker     fileLocker
}

func (p *PortAllocator) AllocatePort(handle string) (int, error) {
	file, err := p.Locker.Open()
	if err != nil {
		return -1, fmt.Errorf("open lock: %s", err)
	}
	defer file.Close() // defer not tested

	pool := &Pool{}
	err = p.Serializer.DecodeAll(file, pool)
	if err != nil {
		return -1, fmt.Errorf("decoding state file: %s", err)
	}

	newPort, err := p.Tracker.AcquireOne(pool)
	if err != nil {
		return -1, fmt.Errorf("acquire port: %s", err)
	}

	err = p.Serializer.EncodeAndOverwrite(file, pool)
	if err != nil {
		return -1, fmt.Errorf("encode and overwrite: %s", err)
	}

	return newPort, nil
}

func (p *PortAllocator) ReleaseAllPorts(handle string) error {
	// - acquire the lock as a file
	// - defer file.Close()

	// - serializer.DecodeAll(file, &pool)
	//    - newPool, newPort, err := tracker.ReleaseMany(pool, allPortsForContainer)
	// - serializer.EncodeAndOverwrite(file, pool)
	return nil
}
