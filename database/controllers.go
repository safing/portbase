package database

import (
	"errors"
	"fmt"
	"sync"

	"github.com/safing/portbase/database/storage"
)

var (
	controllers     = make(map[string]*Controller)
	controllersLock sync.RWMutex
)

func getController(name string) (*Controller, error) {
	if !initialized.IsSet() {
		return nil, errors.New("database not initialized")
	}

	// return database if already started
	controllersLock.RLock()
	controller, ok := controllers[name]
	controllersLock.RUnlock()
	if ok {
		return controller, nil
	}

	controllersLock.Lock()
	defer controllersLock.Unlock()

	if shuttingDown.IsSet() {
		return nil, ErrShuttingDown
	}

	// get db registration
	registeredDB, err := getDatabase(name)
	if err != nil {
		return nil, fmt.Errorf(`could not start database %s: %s`, name, err)
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

	controller = newController(storageInt, registeredDB.ShadowDelete)
	controllers[name] = controller
	return controller, nil
}

// InjectDatabase injects an already running database into the system.
func InjectDatabase(name string, storageInt storage.Interface) (*Controller, error) {
	controllersLock.Lock()
	defer controllersLock.Unlock()

	if shuttingDown.IsSet() {
		return nil, ErrShuttingDown
	}

	_, ok := controllers[name]
	if ok {
		return nil, fmt.Errorf(`database "%s" already loaded`, name)
	}

	registryLock.Lock()
	defer registryLock.Unlock()

	// check if database is registered
	registeredDB, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf(`database "%s" not registered`, name)
	}
	if registeredDB.StorageType != "injected" {
		return nil, fmt.Errorf(`database not of type "injected"`)
	}

	controller := newController(storageInt, false)
	controllers[name] = controller
	return controller, nil
}
