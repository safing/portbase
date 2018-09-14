package database

import (
	"errors"
	"sync"
  "fmt"

  "github.com/Safing/portbase/database/storage"
)

var (
	controllers     = make(map[string]*Controller)
	controllersLock sync.Mutex
)

func getController(name string) (*Controller, error) {
  if !initialized.IsSet() {
    return nil, errors.New("database not initialized")
  }

	controllersLock.Lock()
	defer controllersLock.Unlock()

  // return database if already started
	controller, ok := controllers[name]
	if ok {
    return controller, nil
	}

	// get db registration
	registeredDB, err := getDatabase(name)
	if err != nil {
		return nil, fmt.Errorf(`could not start database %s (type %s): %s`, name, registeredDB.StorageType, err)
	}

	// get location
	dbLocation, err := getLocation(name, registeredDB.StorageType)
	if err != nil {
		return nil, fmt.Errorf(`could not start database %s (type %s): %s`, name, registeredDB.StorageType, err)
	}

  // start database
  storageInt, err := storage.StartDatabase(name, registeredDB.StorageType, dbLocation)
  if err != nil {
    return nil, fmt.Errorf(`could not start database %s (type %s): %s`, name, registeredDB.StorageType, err)
  }

	// create controller
  controller, err = newController(storageInt)
  if err != nil {
    return nil, fmt.Errorf(`could not create controller for database %s: %s`, name, err)
  }

  controllers[name] = controller
  return controller, nil
}

// InjectDatabase injects an already running database into the system.
func InjectDatabase(name string, storageInt storage.Interface) error {
	controllersLock.Lock()
	defer controllersLock.Unlock()

	_, ok := controllers[name]
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

  controller, err := newController(storageInt)
  if err != nil {
    return fmt.Errorf(`could not create controller for database %s: %s`, name, err)
  }

	controllers[name] = controller
	return nil
}
