package sinkhole

import (
	"errors"

	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/database/storage"
)

// Sinkhole is a dummy storage.
type Sinkhole struct {
	name string
}

func init() {
	storage.Register("sinkhole", NewSinkhole)
}

// NewSinkhole creates a dummy database.
func NewSinkhole(name, location string) (storage.Interface, error) {
	return &Sinkhole{
		name: name,
	}, nil
}

// Exists returns whether an entry with the given key exists.
func (s *Sinkhole) Exists(key string) (bool, error) {
	return false, nil
}

// Get returns a database record.
func (s *Sinkhole) Get(key string) (record.Record, error) {
	return nil, storage.ErrNotFound
}

// Put stores a record in the database.
func (s *Sinkhole) Put(m record.Record) error {
	return nil
}

// Delete deletes a record from the database.
func (s *Sinkhole) Delete(key string) error {
	return nil
}

// Query returns a an iterator for the supplied query.
func (s *Sinkhole) Query(q *query.Query) (*iterator.Iterator, error) {
	return nil, errors.New("query not implemented by sinkhole")
}

// ReadOnly returns whether the database is read only.
func (s *Sinkhole) ReadOnly() bool {
	return false
}

// Maintain runs a light maintenance operation on the database.
func (s *Sinkhole) Maintain() error {
	return nil
}

// MaintainThorough runs a thorough maintenance operation on the database.
func (s *Sinkhole) MaintainThorough() (err error) {
	return nil
}

// Shutdown shuts down the database.
func (s *Sinkhole) Shutdown() error {
	return nil
}
