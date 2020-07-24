package database

import (
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
)

// Hook describes a hook.
type Hook interface {
	UsesPreGet() bool
	PreGet(dbKey string) error

	UsesPostGet() bool
	PostGet(r record.Record) (record.Record, error)

	UsesPrePut() bool
	PrePut(r record.Record) (record.Record, error)
}

// RegisteredHook is a registered database hook.
type RegisteredHook struct {
	q *query.Query
	h Hook
}

// RegisterHook registers a hook for records matching the given query in the database.
func RegisterHook(q *query.Query, hook Hook) (*RegisteredHook, error) {
	_, err := q.Check()
	if err != nil {
		return nil, err
	}

	c, err := getController(q.DatabaseName())
	if err != nil {
		return nil, err
	}

	c.readLock.Lock()
	defer c.readLock.Unlock()
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	rh := &RegisteredHook{
		q: q,
		h: hook,
	}
	c.hooks = append(c.hooks, rh)
	return rh, nil
}

// Cancel unhooks the hook.
func (h *RegisteredHook) Cancel() error {
	c, err := getController(h.q.DatabaseName())
	if err != nil {
		return err
	}

	c.readLock.Lock()
	defer c.readLock.Unlock()
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	for key, hook := range c.hooks {
		if hook.q == h.q {
			c.hooks = append(c.hooks[:key], c.hooks[key+1:]...)
			return nil
		}
	}
	return nil
}
