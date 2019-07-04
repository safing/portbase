package database

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/safing/portbase/utils"
	"github.com/tevino/abool"
)

var (
	initialized = abool.NewBool(false)

	shuttingDown   = abool.NewBool(false)
	shutdownSignal = make(chan struct{})
)

// SetLocation sets the location of the database. This is separate from the initialization to provide the location to other modules earlier.
func SetLocation(location string) (ok bool) {
	if !initialized.IsSet() && rootDir == "" {
		rootDir = location
		return true
	}
	return false
}

// Initialize initialized the database
func Initialize() error {
	if initialized.SetToIf(false, true) {

		err := utils.EnsureDirectory(rootDir, 0755)
		if err != nil {
			return fmt.Errorf("could not create/open database directory (%s): %s", rootDir, err)
		}

		err = loadRegistry()
		if err != nil {
			return fmt.Errorf("could not load database registry (%s): %s", filepath.Join(rootDir, registryFileName), err)
		}

		// start registry writer
		go registryWriter()

		return nil
	}
	return errors.New("database already initialized")
}

// Shutdown shuts down the whole database system.
func Shutdown() (err error) {
	if shuttingDown.SetToIf(false, true) {
		close(shutdownSignal)
	} else {
		return
	}

	controllersLock.RLock()
	defer controllersLock.RUnlock()

	for _, c := range controllers {
		err = c.Shutdown()
		if err != nil {
			return
		}
	}
	return
}
