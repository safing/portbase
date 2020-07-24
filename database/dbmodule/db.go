package dbmodule

import (
	"errors"

	"github.com/safing/portbase/database"
	"github.com/safing/portbase/dataroot"
	"github.com/safing/portbase/modules"
	"github.com/safing/portbase/utils"
)

var (
	databaseStructureRoot *utils.DirStructure

	module *modules.Module
)

var errMissingLocation = errors.New("database location not configured")

func init() {
	module = modules.Register("database", prep, start, stop)
}

// SetDatabaseLocation sets the location of the database for initialization. Supply either a path or dir structure.
func SetDatabaseLocation(dirStructureRoot *utils.DirStructure) {
	if databaseStructureRoot == nil {
		databaseStructureRoot = dirStructureRoot
	}
}

func prep() error {
	SetDatabaseLocation(dataroot.Root())
	if databaseStructureRoot == nil {
		return errMissingLocation
	}

	return nil
}

func start() error {
	err := database.Initialize(databaseStructureRoot)
	if err != nil {
		return err
	}

	startMaintenanceTasks()
	return nil
}

func stop() error {
	return database.Shutdown()
}
