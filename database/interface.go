package database

import (
	"errors"
	"fmt"
	"time"

	"github.com/tevino/abool"

	"github.com/bluele/gcache"

	"github.com/safing/portbase/database/accessor"
	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
)

const (
	getDBFromKey = ""
)

// Interface provides a method to access the database with attached options.
type Interface struct {
	options *Options
	cache   gcache.Cache
}

// Options holds options that may be set for an Interface instance.
type Options struct {
	Local                     bool
	Internal                  bool
	AlwaysMakeSecret          bool
	AlwaysMakeCrownjewel      bool
	AlwaysSetRelativateExpiry int64
	AlwaysSetAbsoluteExpiry   int64
	CacheSize                 int
}

// Apply applies options to the record metadata.
func (o *Options) Apply(r record.Record) {
	r.UpdateMeta()
	if o.AlwaysMakeSecret {
		r.Meta().MakeSecret()
	}
	if o.AlwaysMakeCrownjewel {
		r.Meta().MakeCrownJewel()
	}
	if o.AlwaysSetAbsoluteExpiry > 0 {
		r.Meta().SetAbsoluteExpiry(o.AlwaysSetAbsoluteExpiry)
	} else if o.AlwaysSetRelativateExpiry > 0 {
		r.Meta().SetRelativateExpiry(o.AlwaysSetRelativateExpiry)
	}
}

// NewInterface returns a new Interface to the database.
func NewInterface(opts *Options) *Interface {
	if opts == nil {
		opts = &Options{}
	}

	new := &Interface{
		options: opts,
	}
	if opts.CacheSize > 0 {
		new.cache = gcache.New(opts.CacheSize).ARC().Expiration(time.Hour).Build()
	}
	return new
}

func (i *Interface) checkCache(key string) (record.Record, bool) {
	if i.cache != nil {
		cacheVal, err := i.cache.Get(key)
		if err == nil {
			r, ok := cacheVal.(record.Record)
			if ok {
				return r, true
			}
		}
	}
	return nil, false
}

func (i *Interface) updateCache(r record.Record) {
	if i.cache != nil {
		_ = i.cache.Set(r.Key(), r)
	}
}

// Exists return whether a record with the given key exists.
func (i *Interface) Exists(key string) (bool, error) {
	_, _, err := i.getRecord(getDBFromKey, key, false, false)
	if err != nil {
		if err == ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Get return the record with the given key.
func (i *Interface) Get(key string) (record.Record, error) {
	r, ok := i.checkCache(key)
	if ok {
		if !r.Meta().CheckPermission(i.options.Local, i.options.Internal) {
			return nil, ErrPermissionDenied
		}
		return r, nil
	}

	r, _, err := i.getRecord(getDBFromKey, key, true, false)
	return r, err
}

func (i *Interface) getRecord(dbName string, dbKey string, check bool, mustBeWriteable bool) (r record.Record, db *Controller, err error) {
	if dbName == "" {
		dbName, dbKey = record.ParseKey(dbKey)
	}

	db, err = getController(dbName)
	if err != nil {
		return nil, nil, err
	}

	if mustBeWriteable && db.ReadOnly() {
		return nil, nil, ErrReadOnly
	}

	r, err = db.Get(dbKey)
	if err != nil {
		if err == ErrNotFound {
			return nil, db, err
		}
		return nil, nil, err
	}

	if check && !r.Meta().CheckPermission(i.options.Local, i.options.Internal) {
		return nil, nil, ErrPermissionDenied
	}

	return r, db, nil
}

// InsertValue inserts a value into a record.
func (i *Interface) InsertValue(key string, attribute string, value interface{}) error {
	r, db, err := i.getRecord(getDBFromKey, key, true, true)
	if err != nil {
		return err
	}

	r.Lock()
	defer r.Unlock()

	var acc accessor.Accessor
	if r.IsWrapped() {
		wrapper, ok := r.(*record.Wrapper)
		if !ok {
			return errors.New("record is malformed (reports to be wrapped but is not of type *record.Wrapper)")
		}
		acc = accessor.NewJSONBytesAccessor(&wrapper.Data)
	} else {
		acc = accessor.NewStructAccessor(r)
	}

	err = acc.Set(attribute, value)
	if err != nil {
		return fmt.Errorf("failed to set value with %s: %s", acc.Type(), err)
	}

	i.options.Apply(r)
	return db.Put(r)
}

// Put saves a record to the database.
func (i *Interface) Put(r record.Record) (err error) {
	// get record or only database
	var db *Controller
	if !i.options.Internal || !i.options.Local {
		_, db, err = i.getRecord(r.DatabaseName(), r.DatabaseKey(), true, true)
		if err != nil && err != ErrNotFound {
			return err
		}
	} else {
		db, err = getController(r.DatabaseKey())
		if err != nil {
			return err
		}
	}

	r.Lock()
	defer r.Unlock()

	i.options.Apply(r)

	i.updateCache(r)
	return db.Put(r)
}

// PutNew saves a record to the database as a new record (ie. with new timestamps).
func (i *Interface) PutNew(r record.Record) (err error) {
	// get record or only database
	var db *Controller
	if !i.options.Internal || !i.options.Local {
		_, db, err = i.getRecord(r.DatabaseName(), r.DatabaseKey(), true, true)
		if err != nil && err != ErrNotFound {
			return err
		}
	} else {
		db, err = getController(r.DatabaseKey())
		if err != nil {
			return err
		}
	}

	r.Lock()
	defer r.Unlock()

	if r.Meta() != nil {
		r.Meta().Reset()
	}
	i.options.Apply(r)
	i.updateCache(r)
	return db.Put(r)
}

// PutMany stores many records in the database. Warning: This is nearly a direct database access and omits many things:
// - Record locking
// - Hooks
// - Subscriptions
// - Caching
func (i *Interface) PutMany(dbName string) (put func(record.Record) error) {
	interfaceBatch := make(chan record.Record, 100)

	// permission check
	if !i.options.Internal || !i.options.Local {
		return func(r record.Record) error {
			return ErrPermissionDenied
		}
	}

	// get database
	db, err := getController(dbName)
	if err != nil {
		return func(r record.Record) error {
			return err
		}
	}

	// start database access
	dbBatch, errs := db.PutMany()
	finished := abool.New()
	var internalErr error

	// interface options proxy
	go func() {
		defer close(dbBatch) // signify that we are finished
		for {
			select {
			case r := <-interfaceBatch:
				// finished?
				if r == nil {
					return
				}
				// apply options
				i.options.Apply(r)
				// pass along
				dbBatch <- r
			case <-time.After(1 * time.Second):
				// bail out
				internalErr = errors.New("timeout: putmany unused for too long")
				finished.Set()
				return
			}
		}
	}()

	return func(r record.Record) error {
		// finished?
		if finished.IsSet() {
			// check for internal error
			if internalErr != nil {
				return internalErr
			}
			// check for previous error
			select {
			case err := <-errs:
				return err
			default:
				return errors.New("batch is closed")
			}
		}

		// finish?
		if r == nil {
			finished.Set()
			interfaceBatch <- nil // signify that we are finished
			// do not close, as this fn could be called again with nil.
			return <-errs
		}

		// check record scope
		if r.DatabaseName() != dbName {
			return errors.New("record out of database scope")
		}

		// submit
		select {
		case interfaceBatch <- r:
			return nil
		case err := <-errs:
			return err
		}
	}
}

// SetAbsoluteExpiry sets an absolute record expiry.
func (i *Interface) SetAbsoluteExpiry(key string, time int64) error {
	r, db, err := i.getRecord(getDBFromKey, key, true, true)
	if err != nil {
		return err
	}

	r.Lock()
	defer r.Unlock()

	i.options.Apply(r)
	r.Meta().SetAbsoluteExpiry(time)
	return db.Put(r)
}

// SetRelativateExpiry sets a relative (self-updating) record expiry.
func (i *Interface) SetRelativateExpiry(key string, duration int64) error {
	r, db, err := i.getRecord(getDBFromKey, key, true, true)
	if err != nil {
		return err
	}

	r.Lock()
	defer r.Unlock()

	i.options.Apply(r)
	r.Meta().SetRelativateExpiry(duration)
	return db.Put(r)
}

// MakeSecret marks the record as a secret, meaning interfacing processes, such as an UI, are denied access to the record.
func (i *Interface) MakeSecret(key string) error {
	r, db, err := i.getRecord(getDBFromKey, key, true, true)
	if err != nil {
		return err
	}

	r.Lock()
	defer r.Unlock()

	i.options.Apply(r)
	r.Meta().MakeSecret()
	return db.Put(r)
}

// MakeCrownJewel marks a record as a crown jewel, meaning it will only be accessible locally.
func (i *Interface) MakeCrownJewel(key string) error {
	r, db, err := i.getRecord(getDBFromKey, key, true, true)
	if err != nil {
		return err
	}

	r.Lock()
	defer r.Unlock()

	i.options.Apply(r)
	r.Meta().MakeCrownJewel()
	return db.Put(r)
}

// Delete deletes a record from the database.
func (i *Interface) Delete(key string) error {
	r, db, err := i.getRecord(getDBFromKey, key, true, true)
	if err != nil {
		return err
	}

	i.options.Apply(r)
	r.Meta().Delete()
	return db.Put(r)
}

// Query executes the given query on the database.
func (i *Interface) Query(q *query.Query) (*iterator.Iterator, error) {
	_, err := q.Check()
	if err != nil {
		return nil, err
	}

	db, err := getController(q.DatabaseName())
	if err != nil {
		return nil, err
	}

	return db.Query(q, i.options.Local, i.options.Internal)
}

// Subscribe subscribes to updates matching the given query.
func (i *Interface) Subscribe(q *query.Query) (*Subscription, error) {
	_, err := q.Check()
	if err != nil {
		return nil, err
	}

	c, err := getController(q.DatabaseName())
	if err != nil {
		return nil, err
	}

	sub := &Subscription{
		q:        q,
		local:    i.options.Local,
		internal: i.options.Internal,
		Feed:     make(chan record.Record, 1000),
	}
	c.addSubscription(sub)
	return sub, nil
}
