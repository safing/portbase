package database

import (
	"github.com/Safing/portbase/database/query"
	"github.com/Safing/portbase/database/record"
)

// Subscription is a database subscription for updates.
type Subscription struct {
	q    *query.Query
	Feed chan record.Record
	Err  error
}

// Subscribe subscribes to updates matching the given query.
func Subscribe(q *query.Query) (*Subscription, error) {
	_, err := q.Check()
	if err != nil {
		return nil, err
	}

	c, err := getController(q.DatabaseName())
	if err != nil {
		return nil, err
	}

	c.readLock.Lock()
	defer c.readLock.Lock()
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	sub := &Subscription{
		q:    q,
		Feed: make(chan record.Record, 100),
	}
	c.subscriptions = append(c.subscriptions, sub)
	return sub, nil
}

// Cancel cancels the subscription.
func (s *Subscription) Cancel() error {
	c, err := getController(s.q.DatabaseName())
	if err != nil {
		return err
	}

	c.readLock.Lock()
	defer c.readLock.Lock()
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	for key, sub := range c.subscriptions {
		if sub.q == s.q {
			c.subscriptions = append(c.subscriptions[:key], c.subscriptions[key+1:]...)
			return nil
		}
	}
	return nil
}
