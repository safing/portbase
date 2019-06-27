package database

import (
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
)

// Subscription is a database subscription for updates.
type Subscription struct {
	q        *query.Query
	local    bool
	internal bool
	canceled bool

	Feed chan record.Record
	Err  error
}

// Cancel cancels the subscription.
func (s *Subscription) Cancel() error {
	c, err := getController(s.q.DatabaseName())
	if err != nil {
		return err
	}

	c.readLock.Lock()
	defer c.readLock.Unlock()
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	if s.canceled {
		return nil
	}
	s.canceled = true
	close(s.Feed)

	for key, sub := range c.subscriptions {
		if sub.q == s.q {
			c.subscriptions = append(c.subscriptions[:key], c.subscriptions[key+1:]...)
			return nil
		}
	}
	return nil
}
