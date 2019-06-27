package storage

import (
	"errors"

	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
)

var (
	errNotImplemented = errors.New("not implemented")
)

// InjectBase is a dummy base structure to reduce boilerplate code for injected storage interfaces.
type InjectBase struct{}

// Get returns a database record.
func (i *InjectBase) Get(key string) (record.Record, error) {
	return nil, errNotImplemented
}

// Put stores a record in the database.
func (i *InjectBase) Put(m record.Record) error {
	return errNotImplemented
}

// Delete deletes a record from the database.
func (i *InjectBase) Delete(key string) error {
	return errNotImplemented
}

// Query returns a an iterator for the supplied query.
func (i *InjectBase) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {
	return nil, errNotImplemented
}

// ReadOnly returns whether the database is read only.
func (i *InjectBase) ReadOnly() bool {
	return true
}

// Injected returns whether the database is injected.
func (i *InjectBase) Injected() bool {
	return true
}

// Maintain runs a light maintenance operation on the database.
func (i *InjectBase) Maintain() error {
	return nil
}

// MaintainThorough runs a thorough maintenance operation on the database.
func (i *InjectBase) MaintainThorough() error {
	return nil
}

// Shutdown shuts down the database.
func (i *InjectBase) Shutdown() error {
	return nil
}
