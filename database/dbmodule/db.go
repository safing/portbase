package dbmodule

import (
	"errors"

	"github.com/safing/portbase/database"
	"github.com/safing/portbase/modules"
	"github.com/safing/portbase/utils"
)

var (
	databasePath          string
	databaseStructureRoot *utils.DirStructure

	module *modules.Module
)

func init() {
	module = modules.Register("database", prep, start, stop, "base")
}

// SetDatabaseLocation sets the location of the database for initialization. Supply either a path or dir structure.
func SetDatabaseLocation(dirPath string, dirStructureRoot *utils.DirStructure) {
	databasePath = dirPath
	databaseStructureRoot = dirStructureRoot
}

func prep() error {
	if databasePath == "" && databaseStructureRoot == nil {
		return errors.New("no database location specified")
	}
	return nil
}

func start() error {
	err := database.Initialize(databasePath, databaseStructureRoot)
	if err != nil {
		return err
	}

	registerMaintenanceTasks()
	return nil
}

func stop() error {
	return database.Shutdown()
}
