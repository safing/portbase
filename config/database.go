package config

import (
  "errors"
  "sort"
  "strings"

  "github.com/safing/portbase/log"
	"github.com/safing/portbase/database"
  "github.com/safing/portbase/database/storage"
  "github.com/safing/portbase/database/record"
  "github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/iterator"
)

var (
  dbController *database.Controller
)

// ConfigStorageInterface provices a storage.Interface to the configuration manager.
type ConfigStorageInterface struct {
	storage.InjectBase
}

// Get returns a database record.
func (s *ConfigStorageInterface) Get(key string) (record.Record, error) {
  optionsLock.Lock()
  defer optionsLock.Unlock()

  opt, ok := options[key]
  if !ok {
    return nil, storage.ErrNotFound
  }

  return opt.Export()
}

// Put stores a record in the database.
func (s *ConfigStorageInterface) Put(r record.Record) error {
  if r.Meta().Deleted > 0 {
    return setConfigOption(r.DatabaseKey(), nil, false)
  }

  acc := r.GetAccessor(r)
  if acc == nil {
    return errors.New("invalid data")
  }

  val, ok := acc.Get("Value")
  if !ok || val == nil {
    return setConfigOption(r.DatabaseKey(), nil, false)
  }

  optionsLock.RLock()
  option, ok := options[r.DatabaseKey()]
	optionsLock.RUnlock()
  if !ok {
    return errors.New("config option does not exist")
  }

  var value interface{}
  switch option.OptType {
  case   OptTypeString      :
    value, ok = acc.GetString("Value")
  case   OptTypeStringArray :
    value, ok = acc.GetStringArray("Value")
  case   OptTypeInt         :
    value, ok = acc.GetInt("Value")
  case   OptTypeBool        :
    value, ok = acc.GetBool("Value")
  }
  if !ok {
    return errors.New("received invalid value in \"Value\"")
  }

  err := setConfigOption(r.DatabaseKey(), value, false)
  if err != nil {
    return err
  }
  return nil
}

// Delete deletes a record from the database.
func (s *ConfigStorageInterface) Delete(key string) error {
  return setConfigOption(key, nil, false)
}

// Query returns a an iterator for the supplied query.
func (s *ConfigStorageInterface) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {

  optionsLock.Lock()
  defer optionsLock.Unlock()

  it := iterator.New()
  var opts []*Option
  for _, opt := range options {
    if strings.HasPrefix(opt.Key, q.DatabaseKeyPrefix()) {
      opts = append(opts, opt)
    }
  }

  go s.processQuery(q, it, opts)

  return it, nil
}

func (s *ConfigStorageInterface) processQuery(q *query.Query, it *iterator.Iterator, opts []*Option) {

  sort.Sort(sortableOptions(opts))

  for _, opt := range opts {
    r, err := opt.Export()
    if err != nil {
      it.Finish(err)
      return
    }
    it.Next <- r
  }

  it.Finish(nil)
}

// ReadOnly returns whether the database is read only.
func (s *ConfigStorageInterface) ReadOnly() bool {
	return false
}

func registerAsDatabase() error {
	_, err := database.Register(&database.Database{
		Name:        "config",
		Description: "Configuration Manager",
		StorageType: "injected",
		PrimaryAPI:  "",
	})
  if err != nil {
    return err
  }

  controller, err := database.InjectDatabase("config", &ConfigStorageInterface{})
  if err != nil {
    return err
  }

  dbController = controller
	return nil
}

func pushFullUpdate() {
  optionsLock.RLock()
  defer optionsLock.RUnlock()

  for _, option := range options {
    pushUpdate(option)
  }
}

func pushUpdate(option *Option) {
  r, err := option.Export()
  if err != nil {
    log.Errorf("failed to export option to push update: %s", err)
  } else {
    dbController.PushUpdate(r)
  }
}
