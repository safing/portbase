package database

import (
	"sync"
	"time"

	"github.com/tevino/abool"

	"github.com/Safing/portbase/database/iterator"
	"github.com/Safing/portbase/database/query"
	"github.com/Safing/portbase/database/record"
	"github.com/Safing/portbase/database/storage"
)

// A Controller takes care of all the extra database logic.
type Controller struct {
	storage   storage.Interface
	writeLock sync.RWMutex
	readLock  sync.RWMutex
	migrating *abool.AtomicBool
}

// newController creates a new controller for a storage.
func newController(storageInt storage.Interface) (*Controller, error) {
	return &Controller{
		storage:   storageInt,
		migrating: abool.NewBool(false),
	}, nil
}

// Get return the record with the given key.
func (c *Controller) Get(key string) (record.Record, error) {
	r, err := c.storage.Get(key)
	if err != nil {
		return nil, err
	}

	if !r.Meta().CheckValidity(time.Now().Unix()) {
		return nil, ErrNotFound
	}

	return r, nil
}

// Put saves a record in the database.
func (c *Controller) Put(r record.Record) error {
	return c.storage.Put(r)
}

func (c *Controller) Delete(key string) error {
	r, err := c.Get(key)
	if err != nil {
		return err
	}
	r.Meta().Deleted = time.Now().Unix()
	return c.Put(r)
}

// Partial
// What happens if I mutate a value that does not yet exist? How would I know its type?
func (c *Controller) InsertPartial(key string, partialObject interface{}) error {
	return nil
}

func (c *Controller) InsertValue(key string, attribute string, value interface{}) error {
	return nil
}

// Query
func (c *Controller) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {
	return nil, nil
}

// Meta
func (c *Controller) SetAbsoluteExpiry(key string, time int64) error {
	return nil
}

func (c *Controller) SetRelativateExpiry(key string, duration int64) error {
	return nil
}

func (c *Controller) MakeCrownJewel(key string) error {
	return nil
}

func (c *Controller) MakeSecret(key string) error {
	return nil
}
