package dbmodule

import (
	"errors"
	"flag"
	"sync"

	"github.com/Safing/portbase/database"
	"github.com/Safing/portbase/modules"
)

var (
	databaseDir    string
	shutdownSignal = make(chan struct{})
	maintenanceWg  sync.WaitGroup
)

// SetDatabaseLocation sets the location of the database. Must be called before modules.Start and will be overridden by command line options. Intended for unit tests.
func SetDatabaseLocation(location string) {
	databaseDir = location
}

func init() {
	flag.StringVar(&databaseDir, "db", "", "set database directory")

	modules.Register("database", prep, start, stop)
}

func prep() error {
	if databaseDir == "" {
		return errors.New("no database location specified, set with `-db=/path/to/db`")
	}
	ok := database.SetLocation(databaseDir)
	if !ok {
		return errors.New("database location already set")
	}
	return nil
}

func start() error {
	err := database.Initialize()
	if err == nil {
		startMaintainer()
	}
	return err
}

func stop() error {
	close(shutdownSignal)
	maintenanceWg.Wait()
	return database.Shutdown()
}
