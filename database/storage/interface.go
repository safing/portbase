package storage

import (
	"github.com/Safing/portbase/database/iterator"
	"github.com/Safing/portbase/database/query"
	"github.com/Safing/portbase/database/record"
)

// Interface defines the database storage API.
type Interface interface {
	Get(key string) (record.Record, error)
	Put(m record.Record) error
	Delete(key string) error
	Query(q *query.Query, local, internal bool) (*iterator.Iterator, error)

	ReadOnly() bool
	Maintain() error
	MaintainThorough() error
	Shutdown() error
}
