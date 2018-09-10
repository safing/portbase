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

// ReadOnly returns whether the storage is read only.
func (c *Controller) ReadOnly() bool {
	return c.storage.ReadOnly()
}

// Get return the record with the given key.
func (c *Controller) Get(key string) (record.Record, error) {
	if shuttingDown.IsSet() {
		return nil, ErrShuttingDown
	}

	r, err := c.storage.Get(key)
	if err != nil {
		return nil, err
	}

	r.Lock()
	defer r.Unlock()

	if !r.Meta().CheckValidity(time.Now().Unix()) {
		return nil, ErrNotFound
	}

	return r, nil
}

// Put saves a record in the database.
func (c *Controller) Put(r record.Record) error {
	if shuttingDown.IsSet() {
		return ErrShuttingDown
	}

	if c.storage.ReadOnly() {
		return ErrReadOnly
	}

	if r.Meta() == nil {
		r.SetMeta(&record.Meta{})
	}
	r.Meta().Update()

	return c.storage.Put(r)
}

// Query executes the given query on the database.
func (c *Controller) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {
	if shuttingDown.IsSet() {
		return nil, ErrShuttingDown
	}
	return c.storage.Query(q, local, internal)
}

// Maintain runs the Maintain method no the storage.
func (c *Controller) Maintain() error {
	return c.storage.Maintain()
}

// MaintainThorough runs the MaintainThorough method no the storage.
func (c *Controller) MaintainThorough() error {
	return c.storage.MaintainThorough()
}

// Shutdown shuts down the storage.
func (c *Controller) Shutdown() error {
	return c.storage.Shutdown()
}
