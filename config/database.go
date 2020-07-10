package config

import (
	"sort"
	"strings"

	"github.com/safing/portbase/database"
	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/database/storage"
	"github.com/safing/portbase/log"
)

var dbController *database.Controller

// StorageInterface provices a storage.Interface to the configuration manager.
type StorageInterface struct {
	storage.InjectBase
}

// Get returns a database record.
func (s *StorageInterface) Get(key string) (record.Record, error) {
	optionsLock.Lock()
	defer optionsLock.Unlock()

	opt, ok := options[key]
	if !ok {
		return nil, storage.ErrNotFound
	}

	return opt.Export()
}

// Put stores a record in the database.
func (s *StorageInterface) Put(r record.Record) (record.Record, error) {
	if r.Meta().Deleted > 0 {
		return r, setConfigOption(r.DatabaseKey(), nil, false)
	}

	acc := r.GetAccessor(r)
	if acc == nil {
		return nil, ErrInvalidData
	}

	val, ok := acc.Get("Value")
	if !ok || val == nil {
		err := setConfigOption(r.DatabaseKey(), nil, false)
		if err != nil {
			return nil, err
		}
		return s.Get(r.DatabaseKey())
	}

	optionsLock.RLock()
	option, ok := options[r.DatabaseKey()]
	optionsLock.RUnlock()
	if !ok {
		return nil, ErrUnknownOption
	}

	var value interface{}
	switch option.OptType {
	case OptTypeString:
		value, ok = acc.GetString("Value")
	case OptTypeStringArray:
		value, ok = acc.GetStringArray("Value")
	case OptTypeInt:
		value, ok = acc.GetInt("Value")
	case OptTypeBool:
		value, ok = acc.GetBool("Value")
	}
	if !ok {
		val, _ := acc.Get("Value")
		return nil, newInvalidValueError(option.Key, val, "invalid value")
	}

	err := setConfigOption(r.DatabaseKey(), value, false)
	if err != nil {
		return nil, err
	}
	return option.Export()
}

// Delete deletes a record from the database.
func (s *StorageInterface) Delete(key string) error {
	return setConfigOption(key, nil, false)
}

// Query returns a an iterator for the supplied query.
func (s *StorageInterface) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {
	optionsLock.Lock()
	defer optionsLock.Unlock()

	it := iterator.New()
	var opts []*Option
	for _, opt := range options {
		if strings.HasPrefix(opt.Key, q.DatabaseKeyPrefix()) {
			opts = append(opts, opt)
		}
	}

	go s.processQuery(it, opts)

	return it, nil
}

func (s *StorageInterface) processQuery(it *iterator.Iterator, opts []*Option) {
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
func (s *StorageInterface) ReadOnly() bool {
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

	controller, err := database.InjectDatabase("config", &StorageInterface{})
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
