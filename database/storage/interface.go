package storage

import (
	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
)

// Interface defines the database storage API.
type Interface interface {
	Get(key string) (record.Record, error)
	Put(m record.Record) error
	Delete(key string) error
	Query(q *query.Query, local, internal bool) (*iterator.Iterator, error)

	ReadOnly() bool
	Injected() bool
	Maintain() error
	MaintainThorough() error
	Shutdown() error
}
