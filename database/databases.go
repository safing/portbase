package database

import (
	"errors"
	"sync"
  "fmt"
  "path"

  "github.com/Safing/portbase/database/storage"
	"github.com/Safing/portbase/database/record"
)

var (
	databases     = make(map[string]*Controller)
	databasesLock sync.Mutex
)

func splitKeyAndGetDatabase(key string) (dbKey string, db *Controller, err error) {
  var dbName string
  dbName, dbKey = record.ParseKey(key)
  db, err = getDatabase(dbName)
  if err != nil {
    return "", nil, err
  }
  return
}

func getDatabase(name string) (*Controller, error) {
  if !initialized.IsSet() {
    return nil, errors.New("database not initialized")
  }

	databasesLock.Lock()
	defer databasesLock.Unlock()

  // return database if already started
	db, ok := databases[name]
	if ok {
    return db, nil
	}

  registryLock.Lock()
  defer registryLock.Unlock()

  // check if database exists at all
  registeredDB, ok := registry[name]
  if !ok {
    return nil, fmt.Errorf(`database "%s" not registered`, name)
  }

  // start database
  storageInt, err := storage.StartDatabase(name, registeredDB.StorageType, path.Join(rootDir, name, registeredDB.StorageType))
  if err != nil {
    return nil, fmt.Errorf(`could not start database %s (type %s): %s`, name, registeredDB.StorageType, err)
  }

  db, err = newController(storageInt)
  if err != nil {
    return nil, fmt.Errorf(`could not create controller for database %s: %s`, name, err)
  }

  databases[name] = db
  return db, nil
}

// InjectDatabase injects an already running database into the system.
func InjectDatabase(name string, storageInt storage.Interface) error {
	databasesLock.Lock()
	defer databasesLock.Unlock()

	_, ok := databases[name]
	if ok {
		return errors.New(`database "%s" already loaded`)
	}

  registryLock.Lock()
  defer registryLock.Unlock()

  // check if database is registered
  registeredDB, ok := registry[name]
  if !ok {
    return fmt.Errorf(`database "%s" not registered`, name)
  }
  if registeredDB.StorageType != "injected" {
    return fmt.Errorf(`database not of type "injected"`)
  }

  db, err := newController(storageInt)
  if err != nil {
    return fmt.Errorf(`could not create controller for database %s: %s`, name, err)
  }

	databases[name] = db
	return nil
}
