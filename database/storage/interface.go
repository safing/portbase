package storage

import (
	"context"
	"time"

	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
)

// Interface defines the database storage API.
type Interface interface {
	// Primary Interface
	Get(key string) (record.Record, error)
	GetMeta(key string) (*record.Meta, error)
	Put(m record.Record) (record.Record, error)
	Delete(key string) error
	Query(q *query.Query, local, internal bool) (*iterator.Iterator, error)

	// Information and Control
	ReadOnly() bool
	Injected() bool
	Shutdown() error

	// Mandatory Record Maintenance
	MaintainRecordStates(ctx context.Context, purgeDeletedBefore time.Time, shadowDelete bool) error
}

// Maintainer defines the database storage API for backends that require regular maintenance.
type Maintainer interface {
	Maintain(ctx context.Context) error
	MaintainThorough(ctx context.Context) error
}

// Batcher defines the database storage API for backends that support batch operations.
type Batcher interface {
	PutMany(shadowDelete bool) (batch chan<- record.Record, errs <-chan error)
}

// Purger defines the database storage API for backends that support the purge operation.
type Purger interface {
	Purge(ctx context.Context, q *query.Query, local, internal, shadowDelete bool) (int, error)
}
