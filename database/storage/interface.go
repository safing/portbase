package storage

import (
  "github.com/Safing/portbase/database/iterator"
  "github.com/Safing/portbase/database/model"
)

// Interface defines the database storage API.
type Interface interface {
  // Full
  Exists(key string) (bool, error)
  Get(key string) (model.Model, error)
  Create(key string, model model.Model) error
  Update(key string, model model.Model) error // create when not exists
  UpdateOrCreate(key string, model model.Model) error // update, create if not exists.
  Delete(key string) error

  // Partial
  // What happens if I mutate a value that does not yet exist? How would I know its type?
  InsertPartial(key string, partialObject interface{})
  InsertValue(key string, attribute string, value interface{})

  // Query
  Query(*query.Query) (*iterator.Iterator, error)

  // Meta
  LetExpire(key string, timestamp int64) error
  MakeSecret(key string) error // only visible internal
  MakeCrownJewel(key string) error // do not sync between devices
}
