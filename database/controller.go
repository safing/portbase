package database

import (
  "github.com/Safing/portbase/database/record"
)

type Controller struct {
  storage
  writeLock sync.RWMutex
  readLock sync.RWMutex
  migrating *abool.AtomicBool
}

func NewController() (*Controller, error) {

}

	// Retrieve
func (c *Controller) Exists(key string) (bool, error) {}
func (c *Controller) Get(key string) (record.Record, error) {}

// Modify
func (c *Controller) Create(model record.Record) error {}
// create when not exists
func (c *Controller) Update(model record.Record) error {}
// update, create if not exists.
func (c *Controller) UpdateOrCreate(model record.Record) error {}
func (c *Controller) Delete(key string) error {}

// Partial
// What happens if I mutate a value that does not yet exist? How would I know its type?
func (c *Controller) InsertPartial(key string, partialObject interface{}) {}
func (c *Controller) InsertValue(key string, attribute string, value interface{}) {}

// Query
func (c *Controller) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {}

// Meta
func (c *Controller) SetAbsoluteExpiry(key string, time int64) {}
func (c *Controller) SetRelativateExpiry(key string, duration int64) {}
func (c *Controller) MakeCrownJewel(key string) {}
func (c *Controller) MakeSecret(key string) {}
