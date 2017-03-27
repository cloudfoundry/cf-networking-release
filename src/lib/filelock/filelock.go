package filelock

import (
	"os"
	"path/filepath"
	"syscall"
)

//go:generate counterfeiter -o ../fakes/file_locker.go --fake-name FileLocker . FileLocker
type FileLocker interface {
	Open() (*os.File, error)
}

type Locker struct {
	Path string
}

// Open will open and lock a file.  It blocks until the lock is acquired.
// If the file does not yet exist, it creates the file, and any missing
// directories above it in the path.  To release the lock, Close the file.
func (l *Locker) Open() (*os.File, error) {
	dir := filepath.Dir(l.Path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}
	const flags = os.O_RDWR | os.O_CREATE
	file, err := os.OpenFile(l.Path, flags, 0644)
	if err != nil {
		return nil, err
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
	if err != nil {
		return nil, err
	}
	return file, nil
}
