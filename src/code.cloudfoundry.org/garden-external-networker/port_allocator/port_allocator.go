package port_allocator

import (
	"errors"
	"fmt"

	"code.cloudfoundry.org/lib/serial"

	"code.cloudfoundry.org/filelock"
)

//go:generate counterfeiter -o ../fakes/file_locker.go --fake-name FileLocker . FileLocker
type FileLocker interface {
	filelock.FileLocker
}

//go:generate counterfeiter -o ../fakes/locked_file.go --fake-name LockedFile . LockedFile
type LockedFile interface {
	filelock.LockedFile
}

//go:generate counterfeiter -o ../fakes/tracker.go --fake-name Tracker . tracker
type tracker interface {
	AcquireOne(pool *Pool, handle string) (int, error)
	ReleaseAll(pool *Pool, handle string) error
	InRange(port int) bool
}

type PortAllocator struct {
	Tracker    tracker
	Serializer serial.Serializer
	Locker     filelock.FileLocker
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
