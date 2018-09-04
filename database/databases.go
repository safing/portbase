package database


var (
  databases = make(map[string]*storage.Interface)
  databasesLock sync.Mutex
)

func getDatabase(name string) *storage.Interface {
  databasesLock.Lock()
  defer   databasesLock.Unlock()
  storage, ok := databases[name]
  if ok {
    return
  }
}

func databaseExists(name string) (exists bool) {
  // check if folder exists
  return true
}

// CreateDatabase creates a new database with given name and type.
func CreateDatabase(name string, storageType string) error {
  databasesLock.Lock()
  defer   databasesLock.Unlock()
  _, ok := databases[name]
  if ok {
    return errors.New("database with this name already loaded.")
  }
  if databaseExists(name) {
    return errors.New("database with this name already exists.")
  }

  iface, err := startDatabase(name)
  if err != nil {
    return err
  }
  databases[name] = iface
  return nil
}

// InjectDatabase injects an already running database into the system.
func InjectDatabase(name string, iface *storage.Interface) error {
  databasesLock.Lock()
  defer   databasesLock.Unlock()
  _, ok := databases[name]
  if ok {
    return errors.New("database with this name already loaded.")
  }
  if databaseExists(name) {
    return errors.New("database with this name already exists.")
  }
  databases[name] = iface
  return nil
}
