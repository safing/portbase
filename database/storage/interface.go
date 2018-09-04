package storage

import (
	"github.com/Safing/portbase/database/iterator"
	"github.com/Safing/portbase/database/model"
	"github.com/Safing/portbase/database/query"
)

// Interface defines the database storage API.
type Interface interface {
	// Retrieve
	Exists(key string) (bool, error)
	Get(key string) (model.Model, error)

	// Modify
	Create(model model.Model) error
	Update(model model.Model) error         // create when not exists
	UpdateOrCreate(model model.Model) error // update, create if not exists.
	Delete(key string) error

	// Partial
	// What happens if I mutate a value that does not yet exist? How would I know its type?
	InsertPartial(key string, partialObject interface{})
	InsertValue(key string, attribute string, value interface{})

	// Query
	Query(q *query.Query, local, internal bool) (*iterator.Iterator, error)

	// Meta
	SetAbsoluteExpiry(key string, time int64)
	SetRelativateExpiry(key string, duration int64)
	MakeCrownJewel(key string)
	MakeSecret(key string)
}
