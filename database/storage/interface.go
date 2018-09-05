package storage

import (
	"github.com/Safing/portbase/database/iterator"
	"github.com/Safing/portbase/database/record"
	"github.com/Safing/portbase/database/query"
)

// Interface defines the database storage API.
type Interface interface {
	Exists(key string) (bool, error)
	Get(key string) (record.Record, error)
	Put(m record.Record) error
	Delete(key string) error
	Query(q *query.Query) (*iterator.Iterator, error)

	Maintain() error
	MaintainThorough() error
	Shutdown() error
}
