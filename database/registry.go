package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/tevino/abool"
)

// RegisteredDatabase holds information about registered databases
type RegisteredDatabase struct {
	Name        string
	Description string
	StorageType string
	PrimaryAPI  string
}

// Equal returns whether this instance equals another.
func (r *RegisteredDatabase) Equal(o *RegisteredDatabase) bool {
	if r.Name != o.Name ||
		r.Description != o.Description ||
		r.StorageType != o.StorageType ||
		r.PrimaryAPI != o.PrimaryAPI {
		return false
	}
	return true
}

const (
	registryFileName = "databases.json"
)

var (
	initialized = abool.NewBool(false)

	registry     map[string]*RegisteredDatabase
	registryLock sync.Mutex
)

// RegisterDatabase registers a new database.
func RegisterDatabase(new *RegisteredDatabase) error {
	if !initialized.IsSet() {
		return errors.New("database not initialized")
	}

	registryLock.Lock()
	defer registryLock.Unlock()

	registeredDB, ok := registry[new.Name]
	if !ok || !new.Equal(registeredDB) {
		registry[new.Name] = new
		return saveRegistry()
	}

	return nil
}

// Initialize initialized the database
func Initialize(location string) error {
	if initialized.SetToIf(false, true) {
		rootDir = location

		err := checkRootDir()
		if err != nil {
			return fmt.Errorf("could not create/open database directory (%s): %s", rootDir, err)
		}

		err = loadRegistry()
		if err != nil {
			return fmt.Errorf("could not load database registry (%s): %s", path.Join(rootDir, registryFileName), err)
		}

		return nil
	}
	return errors.New("database already initialized")
}

func checkRootDir() error {
	// open dir
	dir, err := os.Open(rootDir)
	if err != nil {
		if err == os.ErrNotExist {
			return os.MkdirAll(rootDir, 0700)
		}
		return err
	}
	defer dir.Close()

	fileInfo, err := dir.Stat()
	if err != nil {
		return err
	}

	if fileInfo.Mode().Perm() != 0700 {
		return dir.Chmod(0700)
	}
	return nil
}

func loadRegistry() error {
	registryLock.Lock()
	defer registryLock.Unlock()

	// read file
	filePath := path.Join(rootDir, registryFileName)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if err == os.ErrNotExist {
			registry = make(map[string]*RegisteredDatabase)
			return nil
		}
		return err
	}

	// parse
	new := make(map[string]*RegisteredDatabase)
	err = json.Unmarshal(data, new)
	if err != nil {
		return err
	}

	// set
	registry = new
	return nil
}

func saveRegistry() error {
	registryLock.Lock()
	defer registryLock.Unlock()

	// marshal
	data, err := json.Marshal(registry)
	if err != nil {
		return err
	}

	// write file
	filePath := path.Join(rootDir, registryFileName)
	return ioutil.WriteFile(filePath, data, 0600)
}
