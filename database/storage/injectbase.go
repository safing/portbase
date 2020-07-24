package storage

import (
	"context"
	"time"

	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
)

// InjectBase is a dummy base structure to reduce boilerplate code for injected storage interfaces.
type InjectBase struct{}

// Get returns a database record.
func (i *InjectBase) Get(key string) (record.Record, error) {
	return nil, ErrNotImplemented
}

// Put stores a record in the database.
func (i *InjectBase) Put(m record.Record) (record.Record, error) {
	return nil, ErrNotImplemented
}

// PutMany stores many records in the database.
func (i *InjectBase) PutMany() (batch chan record.Record, err chan error) {
	batch = make(chan record.Record)
	err = make(chan error, 1)
	err <- ErrNotImplemented
	return
}

// Delete deletes a record from the database.
func (i *InjectBase) Delete(key string) error {
	return ErrNotImplemented
}

// Query returns a an iterator for the supplied query.
func (i *InjectBase) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {
	return nil, ErrNotImplemented
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
func (i *InjectBase) Maintain(ctx context.Context) error {
	return nil
}

// MaintainThorough runs a thorough maintenance operation on the database.
func (i *InjectBase) MaintainThorough(ctx context.Context) error {
	return nil
}

// MaintainRecordStates maintains records states in the database.
func (i *InjectBase) MaintainRecordStates(ctx context.Context, purgeDeletedBefore time.Time) error {
	return nil
}

// Shutdown shuts down the database.
func (i *InjectBase) Shutdown() error {
	return nil
}
