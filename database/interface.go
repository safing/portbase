package database

import (
	"errors"
	"fmt"

	"github.com/Safing/portbase/database/accessor"
	"github.com/Safing/portbase/database/iterator"
	"github.com/Safing/portbase/database/query"
	"github.com/Safing/portbase/database/record"
)

const (
	getDBFromKey = ""
)

// Interface provides a method to access the database with attached options.
type Interface struct {
	options *Options
}

// Options holds options that may be set for an Interface instance.
type Options struct {
	Local                bool
	Internal             bool
	AlwaysMakeSecret     bool
	AlwaysMakeCrownjewel bool
}

// Apply applies options to the record metadata.
func (o *Options) Apply(r record.Record) {
	if r.Meta() == nil {
		r.SetMeta(&record.Meta{})
	}
	if o.AlwaysMakeSecret {
		r.Meta().MakeSecret()
	}
	if o.AlwaysMakeCrownjewel {
		r.Meta().MakeCrownJewel()
	}
}

// NewInterface returns a new Interface to the database.
func NewInterface(opts *Options) *Interface {
	if opts == nil {
		opts = &Options{}
	}

	return &Interface{
		options: opts,
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
func (i *Interface) Put(r record.Record) error {
	_, db, err := i.getRecord(r.DatabaseName(), r.DatabaseKey(), true, true)
	if err != nil && err != ErrNotFound {
		return err
	}

	i.options.Apply(r)
	return db.Put(r)
}

// PutNew saves a record to the database as a new record (ie. with new timestamps).
func (i *Interface) PutNew(r record.Record) error {
	_, db, err := i.getRecord(r.DatabaseName(), r.DatabaseKey(), true, true)
	if err != nil && err != ErrNotFound {
		return err
	}

	i.options.Apply(r)
	r.Meta().Reset()
	return db.Put(r)
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

	r.Lock()
	defer r.Unlock()

	i.options.Apply(r)
	r.Meta().Delete()
	return db.Put(r)
}

// Query executes the given query on the database.
func (i *Interface) Query(q *query.Query) (*iterator.Iterator, error) {
	db, err := getController(q.DatabaseName())
	if err != nil {
		return nil, err
	}

	return db.Query(q, i.options.Local, i.options.Internal)
}
