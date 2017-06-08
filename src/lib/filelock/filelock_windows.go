package filelock

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32     = windows.NewLazySystemDLL("kernel32.dll")
	lockFileEx   = kernel32.NewProc("LockFileEx")
	unlockFileEx = kernel32.NewProc("UnlockFileEx")
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

	h, err := handle(file)
	if err != nil {
		return nil, err
	}

	if err := lockFile(h); err != nil {
		return nil, err
	}

	return &lockedFile{file}, nil
}

func (f *lockedFile) Close() error {
	h, err := handle(f.file)
	if err != nil {
		return err
	}

	if err := unlockFile(h); err != nil {
		return err
	}

	return f.file.Close()
}

func handle(f *os.File) (syscall.Handle, error) {
	h := syscall.Handle(f.Fd())
	if h == syscall.InvalidHandle {
		return h, fmt.Errorf("invalid file descriptor for %s", f.Name())
	}

	return h, nil
}

const LOCKFILE_EXCLUSIVE_LOCK = 2

type overlapped struct {
	internal     uintptr
	internalHigh uintptr
	offest       [8]byte
	handle       windows.Handle
}

func lockFile(h syscall.Handle) error {
	if err := lockFileEx.Find(); err != nil {
		return err
	}

	event, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(event)
	o := &overlapped{handle: event}

	flags := LOCKFILE_EXCLUSIVE_LOCK
	r0, _, err := syscall.Syscall6(lockFileEx.Addr(), 6, uintptr(h), uintptr(flags), 0, math.MaxInt32, math.MaxInt32, uintptr(unsafe.Pointer(o)))
	if int32(r0) == 0 {
		return fmt.Errorf("error locking file: %s", err.Error())
	}

	return nil
}

func unlockFile(h syscall.Handle) error {
	if err := unlockFileEx.Find(); err != nil {
		return err
	}

	event, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(event)
	o := &overlapped{handle: event}

	r0, _, err := syscall.Syscall6(unlockFileEx.Addr(), 5, uintptr(h), 0, math.MaxInt32, math.MaxInt32, uintptr(unsafe.Pointer(o)), 0)
	if int32(r0) == 0 {
		return fmt.Errorf("error unlocking file: %s", err.Error())
	}

	return nil
}
