package database

// Maintain runs the Maintain method on all storages.
func Maintain() (err error) {
  controllers := duplicateControllers()
  for _, c := range controllers {
    err = c.Maintain()
    if err != nil {
      return
    }
  }
  return
}

// MaintainThorough runs the MaintainThorough method on all storages.
func MaintainThorough() (err error) {
  controllers := duplicateControllers()
  for _, c := range controllers {
    err = c.MaintainThorough()
    if err != nil {
      return
    }
  }
  return
}

// Shutdown shuts down the whole database system.
func Shutdown() (err error) {
  shuttingDown.Set()

  controllers := duplicateControllers()
  for _, c := range controllers {
    err = c.Shutdown()
    if err != nil {
      return
    }
  }
  return
}

func duplicateControllers() (controllers []*Controller) {
  databasesLock.Lock()
  defer databasesLock.Unlock()

  for _, c := range databases {
    controllers = append(controllers, c)
  }

  return
}
