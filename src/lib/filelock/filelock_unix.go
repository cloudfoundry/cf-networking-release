// +build !windows

package filelock

import (
	"os"
	"path/filepath"
	"syscall"
)

// Open will open and lock a file.  It blocks until the lock is acquired.
// If the file does not yet exist, it creates the file, and any missing
// directories above it in the path.  To release the lock, Close the file.
func (l *locker) Open() (LockedFile, error) {
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	const flags = os.O_RDWR | os.O_CREATE
	file, err := os.OpenFile(l.path, flags, 0600)
	if err != nil {
		return nil, err
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
	if err != nil {
		return nil, err
	}
	return &lockedFile{file}, nil
}

func (f *lockedFile) Close() error {
	return f.file.Close()
}
