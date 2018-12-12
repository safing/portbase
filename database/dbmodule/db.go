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

func init() {
	flag.StringVar(&databaseDir, "db", "", "set database directory")

	modules.Register("database", prep, start, stop)
}

func prep() error {
	if databaseDir == "" {
		return errors.New("no database location specified, set with `-db=/path/to/db`")
	}
	return nil
}

func start() error {
	err := database.Initialize(databaseDir)
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
