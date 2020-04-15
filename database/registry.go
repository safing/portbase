package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/tevino/abool"
)

const (
	registryFileName = "databases.json"
)

var (
	writeRegistrySoon = abool.NewBool(false)

	registry     map[string]*Database
	registryLock sync.Mutex

	nameConstraint = regexp.MustCompile("^[A-Za-z0-9_-]{4,}$")
)

// Register registers a new database.
// If the database is already registered, only
// the description and the primary API will be
// updated and the effective object will be returned.
func Register(new *Database) (*Database, error) {
	if !initialized.IsSet() {
		return nil, errors.New("database not initialized")
	}

	registryLock.Lock()
	defer registryLock.Unlock()

	registeredDB, ok := registry[new.Name]
	save := false

	if ok {
		// update database
		if registeredDB.Description != new.Description {
			registeredDB.Description = new.Description
			save = true
		}
		if registeredDB.PrimaryAPI != new.PrimaryAPI {
			registeredDB.PrimaryAPI = new.PrimaryAPI
			save = true
		}
	} else {
		// register new database
		if !nameConstraint.MatchString(new.Name) {
			return nil, errors.New("database name must only contain alphanumeric and `_-` characters and must be at least 4 characters long")
		}

		now := time.Now().Round(time.Second)
		new.Registered = now
		new.LastUpdated = now
		new.LastLoaded = time.Time{}

		registry[new.Name] = new
		save = true
	}

	if save {
		if ok {
			registeredDB.Updated()
		}
		err := saveRegistry(false)
		if err != nil {
			return nil, err
		}
	}

	if ok {
		return registeredDB, nil
	}
	return nil, nil
}

func getDatabase(name string) (*Database, error) {
	registryLock.Lock()
	defer registryLock.Unlock()

	registeredDB, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf(`database "%s" not registered`, name)
	}
	if time.Now().Add(-24 * time.Hour).After(registeredDB.LastLoaded) {
		writeRegistrySoon.Set()
	}
	registeredDB.Loaded()

	return registeredDB, nil
}

func loadRegistry() error {
	registryLock.Lock()
	defer registryLock.Unlock()

	// read file
	filePath := path.Join(rootStructure.Path, registryFileName)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			registry = make(map[string]*Database)
			return nil
		}
		return err
	}

	// parse
	new := make(map[string]*Database)
	err = json.Unmarshal(data, &new)
	if err != nil {
		return err
	}

	// set
	registry = new
	return nil
}

func saveRegistry(lock bool) error {
	if lock {
		registryLock.Lock()
		defer registryLock.Unlock()
	}

	// marshal
	data, err := json.MarshalIndent(registry, "", "\t")
	if err != nil {
		return err
	}

	// write file
	// TODO: write atomically (best effort)
	filePath := path.Join(rootStructure.Path, registryFileName)
	return ioutil.WriteFile(filePath, data, 0600)
}

func registryWriter() {
	for {
		select {
		case <-time.After(1 * time.Hour):
			if writeRegistrySoon.SetToIf(true, false) {
				_ = saveRegistry(true)
			}
		case <-shutdownSignal:
			_ = saveRegistry(true)
			return
		}
	}
}
