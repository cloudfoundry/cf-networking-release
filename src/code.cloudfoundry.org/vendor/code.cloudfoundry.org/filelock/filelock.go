package filelock

import "os"

//go:generate counterfeiter -o fakes/file_locker.go --fake-name FileLocker . FileLocker
type FileLocker interface {
	Open() (LockedFile, error)
}

type locker struct {
	path string
}

func NewLocker(path string) FileLocker {
	return &locker{path}
}

//go:generate counterfeiter -o fakes/locked_file.go --fake-name LockedFile . LockedFile
type LockedFile interface {
	Close() error
	Read([]byte) (int, error)
	Truncate(int64) error
	Write([]byte) (int, error)
	Seek(int64, int) (int64, error)
}

type lockedFile struct {
	file *os.File
}

func (f *lockedFile) Read(b []byte) (int, error) {
	return f.file.Read(b)
}

func (f *lockedFile) Truncate(a int64) error {
	return f.file.Truncate(a)
}

func (f *lockedFile) Write(b []byte) (int, error) {
	return f.file.Write(b)
}

func (f *lockedFile) Seek(a int64, b int) (int64, error) {
	return f.file.Seek(a, b)
}
