package database

import (
	"errors"
	"fmt"
	"path"

	"github.com/tevino/abool"
)

var (
	initialized = abool.NewBool(false)

	shuttingDown   = abool.NewBool(false)
	shutdownSignal = make(chan struct{})
)

// Initialize initialized the database
func Initialize(location string) error {
	if initialized.SetToIf(false, true) {
		rootDir = location

		err := ensureDirectory(rootDir)
		if err != nil {
			return fmt.Errorf("could not create/open database directory (%s): %s", rootDir, err)
		}

		err = loadRegistry()
		if err != nil {
			return fmt.Errorf("could not load database registry (%s): %s", path.Join(rootDir, registryFileName), err)
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
	}

	all := duplicateControllers()
	for _, c := range all {
		err = c.Shutdown()
		if err != nil {
			return
		}
	}
	return
}
