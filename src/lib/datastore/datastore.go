package datastore

import (
	"fmt"
	"io/ioutil"
	"lib/serial"
	"net"
	"os"
	"strconv"
	"sync"
)

//go:generate counterfeiter -o ../fakes/locker.go --fake-name Locker . locker
type locker interface {
	Lock() error
	Unlock() error
}

//go:generate counterfeiter -o ../fakes/datastore.go --fake-name Datastore . Datastore
type Datastore interface {
	Add(handle, ip string, metadata map[string]interface{}) error
	Delete(handle string) (Container, error)
	ReadAll() (map[string]Container, error)
}

type Container struct {
	Handle   string                 `json:"handle"`
	IP       string                 `json:"ip"`
	Metadata map[string]interface{} `json:"metadata"`
}

type Store struct {
	Serializer      serial.Serializer
	Locker          locker
	DataFilePath    string
	VersionFilePath string
	CacheMutex      *sync.RWMutex
	cachedVersion   int
	cachedPool      map[string]Container
}

func validate(handle, ip string) error {
	if handle == "" {
		return fmt.Errorf("invalid handle")
	}

	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid ip: %v", ip)
	}
	return nil
}

func (c *Store) Add(handle, ip string, metadata map[string]interface{}) error {
	if err := validate(handle, ip); err != nil {
		return err
	}

	err := c.Locker.Lock()
	if err != nil {
		return fmt.Errorf("lock: %s", err)
	}
	defer c.Locker.Unlock()

	dataFile, err := os.OpenFile(c.DataFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open data file: %s", err)
	}
	defer dataFile.Close()

	pool := make(map[string]Container)
	err = c.Serializer.DecodeAll(dataFile, &pool)
	if err != nil {
		return fmt.Errorf("decoding file: %s", err)
	}

	pool[handle] = Container{
		Handle:   handle,
		IP:       ip,
		Metadata: metadata,
	}

	err = c.Serializer.EncodeAndOverwrite(dataFile, pool)
	if err != nil {
		return fmt.Errorf("encode and overwrite: %s", err)
	}

	err = c.updateVersion()
	if err != nil {
		return err
	}

	return nil
}

func (c *Store) Delete(handle string) (Container, error) {
	deleted := Container{}
	if handle == "" {
		return deleted, fmt.Errorf("invalid handle")
	}

	err := c.Locker.Lock()
	if err != nil {
		return deleted, fmt.Errorf("lock: %s", err)
	}
	defer c.Locker.Unlock()

	dataFile, err := os.OpenFile(c.DataFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return deleted, fmt.Errorf("open data file: %s", err)
	}
	defer dataFile.Close()

	pool := make(map[string]Container)
	err = c.Serializer.DecodeAll(dataFile, &pool)
	if err != nil {
		return deleted, fmt.Errorf("decoding file: %s", err)
	}

	deleted = pool[handle]

	delete(pool, handle)

	err = c.Serializer.EncodeAndOverwrite(dataFile, pool)
	if err != nil {
		return deleted, fmt.Errorf("encode and overwrite: %s", err)
	}

	err = c.updateVersion()
	if err != nil {
		return deleted, err
	}

	return deleted, nil
}

func (c *Store) ReadAll() (map[string]Container, error) {
	currentVersion, err := c.currentVersion()
	if err != nil {
		return nil, err
	}

	c.CacheMutex.RLock()
	if currentVersion == c.cachedVersion {
		pool := c.cachedPool
		c.CacheMutex.RUnlock()
		return pool, nil
	}
	c.CacheMutex.RUnlock()

	err = c.Locker.Lock()
	if err != nil {
		return nil, fmt.Errorf("lock: %s", err)
	}
	defer c.Locker.Unlock()

	dataFile, err := os.OpenFile(c.DataFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("open data file: %s", err)
	}
	defer dataFile.Close()

	pool := make(map[string]Container)
	err = c.Serializer.DecodeAll(dataFile, &pool)
	if err != nil {
		return nil, fmt.Errorf("decoding file: %s", err)
	}

	// untested
	// we want to get the version again while we have the store locked
	currentVersion, err = c.currentVersion()
	if err != nil {
		return nil, err
	}

	c.CacheMutex.Lock()
	defer c.CacheMutex.Unlock()
	c.cachedPool = pool
	c.cachedVersion = currentVersion

	return pool, nil
}

func (c *Store) updateVersion() error {
	version, err := c.currentVersion()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(c.VersionFilePath, []byte(strconv.Itoa(version+1)), os.ModePerm)
	if err != nil {
		// not tested
		return fmt.Errorf("write version file: %s", err)
	}

	return nil
}

func (c *Store) currentVersion() (int, error) {
	version := 1
	versionFile, err := os.OpenFile(c.VersionFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return version, fmt.Errorf("open version file: %s", err)
	}
	defer versionFile.Close()

	versionContents, err := ioutil.ReadAll(versionFile)
	if err != nil {
		// not tested
		return version, fmt.Errorf("open version file: %s", err)
	}

	if string(versionContents) != "" {
		version, err = strconv.Atoi(string(versionContents))
		if err != nil {
			return version, fmt.Errorf("version file: '%s' is not a number", versionContents)
		}
	}
	return version, err
}
