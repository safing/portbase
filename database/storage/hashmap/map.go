package hashmap

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/database/storage"
)

// HashMap storage.
type HashMap struct {
	name   string
	db     map[string]record.Record
	dbLock sync.RWMutex
}

func init() {
	_ = storage.Register("hashmap", NewHashMap)
}

// NewHashMap creates a hashmap database.
func NewHashMap(name, location string) (storage.Interface, error) {
	return &HashMap{
		name: name,
		db:   make(map[string]record.Record),
	}, nil
}

// Get returns a database record.
func (hm *HashMap) Get(key string) (record.Record, error) {
	hm.dbLock.RLock()
	defer hm.dbLock.RUnlock()

	r, ok := hm.db[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return r, nil
}

// Put stores a record in the database.
func (hm *HashMap) Put(r record.Record) error {
	hm.dbLock.Lock()
	defer hm.dbLock.Unlock()

	hm.db[r.DatabaseKey()] = r
	return nil
}

// Delete deletes a record from the database.
func (hm *HashMap) Delete(key string) error {
	hm.dbLock.Lock()
	defer hm.dbLock.Unlock()

	delete(hm.db, key)
	return nil
}

// Query returns a an iterator for the supplied query.
func (hm *HashMap) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {
	_, err := q.Check()
	if err != nil {
		return nil, fmt.Errorf("invalid query: %s", err)
	}

	queryIter := iterator.New()

	go hm.queryExecutor(queryIter, q, local, internal)
	return queryIter, nil
}

func (hm *HashMap) queryExecutor(queryIter *iterator.Iterator, q *query.Query, local, internal bool) {
	hm.dbLock.RLock()
	defer hm.dbLock.RUnlock()

	var err error

mapLoop:
	for key, record := range hm.db {

		switch {
		case !q.MatchesKey(key):
			continue
		case !q.MatchesRecord(record):
			continue
		case !record.Meta().CheckValidity():
			continue
		case !record.Meta().CheckPermission(local, internal):
			continue
		}

		select {
		case <-queryIter.Done:
			break mapLoop
		case queryIter.Next <- record:
		default:
			select {
			case <-queryIter.Done:
				break mapLoop
			case queryIter.Next <- record:
			case <-time.After(1 * time.Second):
				err = errors.New("query timeout")
				break mapLoop
			}
		}

	}

	queryIter.Finish(err)
}

// ReadOnly returns whether the database is read only.
func (hm *HashMap) ReadOnly() bool {
	return false
}

// Injected returns whether the database is injected.
func (hm *HashMap) Injected() bool {
	return false
}

// Maintain runs a light maintenance operation on the database.
func (hm *HashMap) Maintain() error {
	return nil
}

// MaintainThorough runs a thorough maintenance operation on the database.
func (hm *HashMap) MaintainThorough() (err error) {
	return nil
}

// Shutdown shuts down the database.
func (hm *HashMap) Shutdown() error {
	return nil
}
