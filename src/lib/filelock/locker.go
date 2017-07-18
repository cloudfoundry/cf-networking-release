package filelock

import (
	"fmt"
	"sync"
)

type Locker struct {
	FileLocker FileLocker
	Mutex      *sync.Mutex
	f          LockedFile
}

func (l *Locker) Lock() error {
	l.Mutex.Lock()

	var err error
	l.f, err = l.FileLocker.Open()
	if err != nil {
		l.Mutex.Unlock()
		return fmt.Errorf("open lock file: %s", err)
	}
	return nil
}

func (l *Locker) Unlock() error {
	defer l.Mutex.Unlock()
	return l.f.Close()
}
