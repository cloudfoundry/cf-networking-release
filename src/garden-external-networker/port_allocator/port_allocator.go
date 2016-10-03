package port_allocator

import (
	"errors"
	"fmt"
	"io"
	"os"
)

//go:generate counterfeiter -o ../fakes/tracker.go --fake-name Tracker . tracker
type tracker interface {
	AcquireOne(pool *Pool, handle string) (int, error)
	ReleaseAll(pool *Pool, handle string) error
	InRange(port int) bool
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

func (p *PortAllocator) AllocatePort(handle string, port int) (int, error) {
	if port != 0 {
		if p.Tracker.InRange(port) {
			return -1, errors.New("cannot specify port from allocation range")
		} else {
			return port, nil
		}
	}

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

	newPort, err := p.Tracker.AcquireOne(pool, handle)
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
	file, err := p.Locker.Open()
	if err != nil {
		return fmt.Errorf("open lock: %s", err)
	}
	defer file.Close() // defer not tested

	pool := &Pool{}
	err = p.Serializer.DecodeAll(file, pool)
	if err != nil {
		return fmt.Errorf("decoding state file: %s", err)
	}

	if err := p.Tracker.ReleaseAll(pool, handle); err != nil {
		return fmt.Errorf("release all ports: %s", err)
	}

	err = p.Serializer.EncodeAndOverwrite(file, pool)
	if err != nil {
		return fmt.Errorf("encode and overwrite: %s", err)
	}

	return nil
}
