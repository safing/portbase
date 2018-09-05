package badger

import (
	"errors"
	"time"

	"github.com/dgraph-io/badger"

	"github.com/Safing/portbase/database/iterator"
	"github.com/Safing/portbase/database/record"
	"github.com/Safing/portbase/database/query"
	"github.com/Safing/portbase/database/storage"
)

// Badger database made pluggable for portbase.
type Badger struct {
	name string
	db   *badger.DB
}

func init() {
	storage.Register("badger", NewBadger)
}

// NewBadger opens/creates a badger database.
func NewBadger(name, location string) (storage.Interface, error) {
	opts := badger.DefaultOptions
	opts.Dir = location
	opts.ValueDir = location

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &Badger{
		name: name,
		db:   db,
	}, nil
}

// Exists returns whether an entry with the given key exists.
func (b *Badger) Exists(key string) (bool, error) {
	err := b.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}
		return nil
	})
	if err == nil {
		return true, nil
	}
	return false, nil
}

// Get returns a database record.
func (b *Badger) Get(key string) (record.Record, error) {
	var item *badger.Item

	err := b.db.View(func(txn *badger.Txn) error {
		var err error
		item, err = txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return storage.ErrNotFound
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if item.IsDeletedOrExpired() {
		return nil, storage.ErrNotFound
	}

	data, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	m, err := model.NewRawWrapper(b.name, string(item.Key()), data)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Put stores a record in the database.
func (b *Badger) Put(m record.Record) error {
	data, err := m.MarshalRecord()
	if err != nil {
		return err
	}

	err = b.db.Update(func(txn *badger.Txn) error {
		if m.Meta().GetAbsoluteExpiry() > 0 {
			txn.SetWithTTL([]byte(m.DatabaseKey()), data, time.Duration(m.Meta().GetRelativeExpiry()))
		} else {
			txn.Set([]byte(m.DatabaseKey()), data)
		}
		return nil
	})
	return err
}

// Delete deletes a record from the database.
func (b *Badger) Delete(key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		return nil
	})
}

// Query returns a an iterator for the supplied query.
func (b *Badger) Query(q *query.Query) (*iterator.Iterator, error) {
	return nil, errors.New("query not implemented by badger")
}

// Maintain runs a light maintenance operation on the database.
func (b *Badger) Maintain() error {
	b.db.RunValueLogGC(0.7)
	return nil
}

// MaintainThorough runs a thorough maintenance operation on the database.
func (b *Badger) MaintainThorough() (err error) {
	for err == nil {
		err = b.db.RunValueLogGC(0.7)
	}
	return nil
}

// Shutdown shuts down the database.
func (b *Badger) Shutdown() error {
	return b.db.Close()
}
