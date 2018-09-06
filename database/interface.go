package database

import (
	"github.com/Safing/portbase/database/record"
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
	_, err := i.getRecord(key)
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
	r, err := i.getRecord(key)
	if err != nil {
		return nil, err
	}

	if !r.Meta().CheckPermission(i.options.Local, i.options.Internal) {
		return nil, ErrPermissionDenied
	}

	return r, nil
}

func (i *Interface) getRecord(key string) (record.Record, error) {
	dbKey, db, err := splitKeyAndGetDatabase(key)
	if err != nil {
		return nil, err
	}

	r, err := db.Get(dbKey)
	if err != nil {
		return nil, err
	}

	return r, nil
}
