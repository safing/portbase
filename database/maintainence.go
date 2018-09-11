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
  all := duplicateControllers()
  for _, c := range all {
    err = c.MaintainThorough()
    if err != nil {
      return
    }
  }
  return
}

func duplicateControllers() (all []*Controller) {
  controllersLock.Lock()
  defer controllersLock.Unlock()

  for _, c := range controllers {
    all = append(all, c)
  }

  return
}
