package database

import (
	"sync"

	"github.com/tevino/abool"

	"github.com/Safing/portbase/database/iterator"
	"github.com/Safing/portbase/database/query"
	"github.com/Safing/portbase/database/record"
	"github.com/Safing/portbase/database/storage"
)

// A Controller takes care of all the extra database logic.
type Controller struct {
	storage storage.Interface

	hooks         []*RegisteredHook
	subscriptions []*Subscription

	writeLock sync.RWMutex
	//  Lock: nobody may write
	// RLock: concurrent writing
	readLock sync.RWMutex
	//  Lock: nobody may read
	// RLock: concurrent reading

	migrating   *abool.AtomicBool // TODO
	hibernating *abool.AtomicBool // TODO
}

// newController creates a new controller for a storage.
func newController(storageInt storage.Interface) (*Controller, error) {
	return &Controller{
		storage:     storageInt,
		migrating:   abool.NewBool(false),
		hibernating: abool.NewBool(false),
	}, nil
}

// ReadOnly returns whether the storage is read only.
func (c *Controller) ReadOnly() bool {
	return c.storage.ReadOnly()
}

// Injected returns whether the storage is injected.
func (c *Controller) Injected() bool {
	return c.storage.Injected()
}

// Get return the record with the given key.
func (c *Controller) Get(key string) (record.Record, error) {
	if shuttingDown.IsSet() {
		return nil, ErrShuttingDown
	}

	c.readLock.RLock()
	defer c.readLock.RUnlock()

	// process hooks
	for _, hook := range c.hooks {
		if hook.h.UsesPreGet() && hook.q.MatchesKey(key) {
			err := hook.h.PreGet(key)
			if err != nil {
				return nil, err
			}
		}
	}

	r, err := c.storage.Get(key)
	if err != nil {
		// replace not found error
		if err == storage.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	r.Lock()
	defer r.Unlock()

	// process hooks
	for _, hook := range c.hooks {
		if hook.h.UsesPostGet() && hook.q.Matches(r) {
			r, err = hook.h.PostGet(r)
			if err != nil {
				return nil, err
			}
		}
	}

	if !r.Meta().CheckValidity() {
		return nil, ErrNotFound
	}

	return r, nil
}

// Put saves a record in the database.
func (c *Controller) Put(r record.Record) (err error) {
	if shuttingDown.IsSet() {
		return ErrShuttingDown
	}

	if c.ReadOnly() {
		return ErrReadOnly
	}

	// process hooks
	for _, hook := range c.hooks {
		if hook.h.UsesPrePut() && hook.q.Matches(r) {
			r, err = hook.h.PrePut(r)
			if err != nil {
				return err
			}
		}
	}

	if r.Meta() == nil {
		r.SetMeta(&record.Meta{})
	}
	r.Meta().Update()

	c.writeLock.RLock()
	defer c.writeLock.RUnlock()

	err = c.storage.Put(r)
	if err != nil {
		return err
	}

	// process subscriptions
	for _, sub := range c.subscriptions {
		if r.Meta().CheckPermission(sub.local, sub.internal) && sub.q.Matches(r) {
			select {
			case sub.Feed <- r:
			default:
			}
		}
	}

	return nil
}

// Query executes the given query on the database.
func (c *Controller) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {
	if shuttingDown.IsSet() {
		return nil, ErrShuttingDown
	}

	c.readLock.RLock()
	it, err := c.storage.Query(q, local, internal)
	if err != nil {
		c.readLock.RUnlock()
		return nil, err
	}

	go c.readUnlockerAfterQuery(it)
	return it, nil
}

// PushUpdate pushes a record update to subscribers.
func (c *Controller) PushUpdate(r record.Record) {
	if c != nil {
		c.readLock.RLock()
		defer c.readLock.RUnlock()

		for _, sub := range c.subscriptions {
			if r.Meta().CheckPermission(sub.local, sub.internal) && sub.q.Matches(r) {
				select {
				case sub.Feed <- r:
				default:
				}
			}
		}
	}
}

func (c *Controller) addSubscription(sub *Subscription) {
	c.readLock.Lock()
	defer c.readLock.Unlock()
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	c.subscriptions = append(c.subscriptions, sub)
}

func (c *Controller) readUnlockerAfterQuery(it *iterator.Iterator) {
	<-it.Done
	c.readLock.RUnlock()
}

// Maintain runs the Maintain method on the storage.
func (c *Controller) Maintain() error {
	c.writeLock.RLock()
	defer c.writeLock.RUnlock()
	return c.storage.Maintain()
}

// MaintainThorough runs the MaintainThorough method on the storage.
func (c *Controller) MaintainThorough() error {
	c.writeLock.RLock()
	defer c.writeLock.RUnlock()
	return c.storage.MaintainThorough()
}

// Shutdown shuts down the storage.
func (c *Controller) Shutdown() error {
	// TODO: should we wait for gets/puts/queries to complete?
	return c.storage.Shutdown()
}
