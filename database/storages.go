package database

// A Factory creates a new database of it's type.
type Factory func(name, location string) (*storage.Interface, error)

var (
  storages map[string]Factory
  storagesLock sync.Mutex
)

// RegisterStorage registers a new storage type.
func RegisterStorage(name string, factory Factory) error {
  storagesLock.Lock()
  defer storagesLock.Unlock()

  _, ok := storages[name]
  if ok {
    return errors.New("factory for this type already exists")
  }

  storages[name] = factory
  return nil
}

// startDatabase starts a new database with the given name, storageType at location.
func startDatabase(name, storageType, location string) (*storage.Interface, error) {
  storagesLock.Lock()
  defer storagesLock.Unlock()

  factory, ok := storages[name]
  if !ok {
    return fmt.Errorf("storage of this type (%s) does not exist", storageType)
  }

  return factory(name, location)
}
