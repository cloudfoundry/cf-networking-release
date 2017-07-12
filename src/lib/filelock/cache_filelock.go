package filelock

import (
	"os"
	"time"
	"bytes"
	"io/ioutil"
	"fmt"
)

type CacheFileLock struct {
	fileLocker      FileLocker
	fileLockPath    string
	fileLockModTime time.Time
	cacheFile       []byte
}

func NewCacheFileLock(fileLocker FileLocker, fileLockPath string) *CacheFileLock {
	return &CacheFileLock{
		fileLocker:   fileLocker,
		fileLockPath: fileLockPath,
	}
}

type InMemoryLockedFile struct {
	*bytes.Reader
}

func (InMemoryLockedFile) Close() error {
	return nil
}

func (InMemoryLockedFile) Truncate(int64) error {
	return nil
}

func (InMemoryLockedFile) Write([]byte) (int, error) {
	panic("Not Implemented")
}

func (c *CacheFileLock) Open() (LockedFile, error) {
	fileInfo, err := os.Stat(c.fileLockPath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %s", err)
	}

	if c.fileLockModTime.Before(fileInfo.ModTime()) {
		lockedFile, err := c.fileLocker.Open()
		if err != nil {
			return nil, fmt.Errorf("open file lock: %s", err)
		}
		lockedFileContents, err := ioutil.ReadAll(lockedFile)
		if err != nil {
			return nil, fmt.Errorf("read locked file: %s", err)
		}
		c.fileLockModTime = fileInfo.ModTime()
		c.cacheFile = lockedFileContents
	}

	return InMemoryLockedFile{bytes.NewReader(c.cacheFile)}, nil
}
