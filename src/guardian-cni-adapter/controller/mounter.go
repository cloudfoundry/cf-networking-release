package controller

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type Mounter struct{}

func (m *Mounter) IdempotentlyMount(source, target string) error {
	err := os.MkdirAll(filepath.Dir(target), 0600)
	if err != nil {
		return fmt.Errorf("os.MkdirAll failed: %s", err)
	}

	fd, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("os.Create failed: %s", err)
	}
	defer fd.Close()

	err = unix.Mount(source, target, "none", unix.MS_BIND, "")
	if err != nil {
		return fmt.Errorf("mount failed: %s", err)
	}

	return nil
}

func (m *Mounter) RemoveMount(target string) error {
	err := unix.Unmount(target, unix.MNT_DETACH)
	if err != nil {
		return fmt.Errorf("unmount failed: %s", err)
	}

	err = os.RemoveAll(target)
	if err != nil {
		return fmt.Errorf("removeall failed: %s", err) // not tested
	}

	return nil
}
