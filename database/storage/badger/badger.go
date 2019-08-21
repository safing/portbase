package badger

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger"

	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/database/storage"
	"github.com/safing/portbase/log"
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
	opts := badger.DefaultOptions(location)

	db, err := badger.Open(opts)
	if err == badger.ErrTruncateNeeded {
		// clean up after crash
		log.Warningf("database/storage: truncating corrupted value log of badger database %s: this may cause data loss", name)
		opts.Truncate = true
		db, err = badger.Open(opts)
	}
	if err != nil {
		return nil, err
	}

	return &Badger{
		name: name,
		db:   db,
	}, nil
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

	// return err if deleted or expired
	if item.IsDeletedOrExpired() {
		return nil, storage.ErrNotFound
	}

	data, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	m, err := record.NewRawWrapper(b.name, string(item.Key()), data)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Put stores a record in the database.
func (b *Badger) Put(r record.Record) error {
	data, err := r.MarshalRecord(r)
	if err != nil {
		return err
	}

	err = b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(r.DatabaseKey()), data)
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
func (b *Badger) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {
	_, err := q.Check()
	if err != nil {
		return nil, fmt.Errorf("invalid query: %s", err)
	}

	queryIter := iterator.New()

	go b.queryExecutor(queryIter, q, local, internal)
	return queryIter, nil
}

func (b *Badger) queryExecutor(queryIter *iterator.Iterator, q *query.Query, local, internal bool) {
	err := b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(q.DatabaseKeyPrefix())
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			var data []byte
			err := item.Value(func(val []byte) error {
				data = val
				return nil
			})
			if err != nil {
				return err
			}

			r, err := record.NewRawWrapper(b.name, string(item.Key()), data)
			if err != nil {
				return err
			}

			if !r.Meta().CheckValidity() {
				continue
			}
			if !r.Meta().CheckPermission(local, internal) {
				continue
			}

			if q.MatchesRecord(r) {
				copiedData, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				new, err := record.NewRawWrapper(b.name, r.DatabaseKey(), copiedData)
				if err != nil {
					return err
				}
				select {
				case <-queryIter.Done:
					return nil
				case queryIter.Next <- new:
				default:
					select {
					case queryIter.Next <- new:
					case <-queryIter.Done:
						return nil
					case <-time.After(1 * time.Minute):
						return errors.New("query timeout")
					}
				}
			}

		}
		return nil
	})

	queryIter.Finish(err)
}

// ReadOnly returns whether the database is read only.
func (b *Badger) ReadOnly() bool {
	return false
}

// Injected returns whether the database is injected.
func (b *Badger) Injected() bool {
	return false
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
