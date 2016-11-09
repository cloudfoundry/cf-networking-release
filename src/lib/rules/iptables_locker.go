package rules

import (
	"fmt"
	"lib/filelock"
	"os"
)

type IPTablesLocker struct {
	FileLocker filelock.FileLocker
	f          *os.File
}

// TODO improve test coverage / add a close function to filelocker
func (l *IPTablesLocker) Lock() error {
	var err error
	l.f, err = l.FileLocker.Open()
	if err != nil {
		return fmt.Errorf("open lock file: %s", err)
	}
	return nil
}

func (l *IPTablesLocker) Unlock() error {
	return l.f.Close()
}
