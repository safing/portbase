package database

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/tevino/abool"

	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/database/storage"
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
func newController(storageInt storage.Interface) *Controller {
	return &Controller{
		storage:     storageInt,
		migrating:   abool.NewBool(false),
		hibernating: abool.NewBool(false),
	}
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
	c.readLock.RLock()
	defer c.readLock.RUnlock()

	if shuttingDown.IsSet() {
		return nil, ErrShuttingDown
	}

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
	c.writeLock.RLock()
	defer c.writeLock.RUnlock()

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

	r, err = c.storage.Put(r)
	if err != nil {
		return err
	}
	if r == nil {
		return errors.New("storage returned nil record after successful put operation")
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

// PutMany stores many records in the database.
func (c *Controller) PutMany() (chan<- record.Record, <-chan error) {
	c.writeLock.RLock()
	defer c.writeLock.RUnlock()

	if shuttingDown.IsSet() {
		errs := make(chan error, 1)
		errs <- ErrShuttingDown
		return make(chan record.Record), errs
	}

	if c.ReadOnly() {
		errs := make(chan error, 1)
		errs <- ErrReadOnly
		return make(chan record.Record), errs
	}

	if batcher, ok := c.storage.(storage.Batcher); ok {
		return batcher.PutMany()
	}

	errs := make(chan error, 1)
	errs <- ErrNotImplemented
	return make(chan record.Record), errs
}

// Query executes the given query on the database.
func (c *Controller) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {
	c.readLock.RLock()

	if shuttingDown.IsSet() {
		c.readLock.RUnlock()
		return nil, ErrShuttingDown
	}

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

		if shuttingDown.IsSet() {
			return
		}

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

	if shuttingDown.IsSet() {
		return
	}

	c.subscriptions = append(c.subscriptions, sub)
}

func (c *Controller) readUnlockerAfterQuery(it *iterator.Iterator) {
	<-it.Done
	c.readLock.RUnlock()
}

// Maintain runs the Maintain method on the storage.
func (c *Controller) Maintain(ctx context.Context) error {
	c.writeLock.RLock()
	defer c.writeLock.RUnlock()

	if shuttingDown.IsSet() {
		return nil
	}

	return c.storage.Maintain(ctx)
}

// MaintainThorough runs the MaintainThorough method on the storage.
func (c *Controller) MaintainThorough(ctx context.Context) error {
	c.writeLock.RLock()
	defer c.writeLock.RUnlock()

	if shuttingDown.IsSet() {
		return nil
	}

	return c.storage.MaintainThorough(ctx)
}

// MaintainRecordStates runs the record state lifecycle maintenance on the storage.
func (c *Controller) MaintainRecordStates(ctx context.Context, purgeDeletedBefore time.Time) error {
	c.writeLock.RLock()
	defer c.writeLock.RUnlock()

	if shuttingDown.IsSet() {
		return nil
	}

	return c.storage.MaintainRecordStates(ctx, purgeDeletedBefore)
}

// Shutdown shuts down the storage.
func (c *Controller) Shutdown() error {
	// acquire full locks
	c.readLock.Lock()
	defer c.readLock.Unlock()
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	return c.storage.Shutdown()
}
