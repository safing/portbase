package config

import (
	"os"
	"path"

	"github.com/safing/portbase/database"
	"github.com/safing/portbase/modules"
)

func init() {
	modules.Register("config", prep, start, nil, "database")
}

func prep() error {
	return nil
}

func start() error {
	configFilePath = path.Join(database.GetDatabaseRoot(), "config.json")

	err := registerAsDatabase()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	err = loadConfig()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
