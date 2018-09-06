package dbmodule

import (
  "github.com/Safing/portbase/database"
)

var (
	databaseDir string
)

func init() {
  flag.StringVar(&databaseDir, "db", "", "set database directory")

  modules.Register("database", prep, start, stop)
}

func prep() error {
  if databaseDir == "" {
    return errors.New("no database location specified, set with `-db=/path/to/db`")
  }
}

func start() error {
  return database.Initialize(databaseDir)
}

func stop() {
  return database.Shutdown()
}
