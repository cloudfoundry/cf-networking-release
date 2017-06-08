package rules

import (
	"fmt"
	"lib/filelock"
	"sync"
)

type IPTablesLocker struct {
	FileLocker filelock.FileLocker
	f          filelock.LockedFile
	Mutex      *sync.Mutex
}

// TODO improve test coverage / add a close function to filelocker
func (l *IPTablesLocker) Lock() error {
	l.Mutex.Lock()

	var err error
	l.f, err = l.FileLocker.Open()
	if err != nil {
		l.Mutex.Unlock()
		return fmt.Errorf("open lock file: %s", err)
	}
	return nil
}

func (l *IPTablesLocker) Unlock() error {
	defer l.Mutex.Unlock()
	return l.f.Close()
}
