package database

import (
	"time"

	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
)

// Maintain runs the Maintain method on all storages.
func Maintain() (err error) {
	controllers := duplicateControllers()
	for _, c := range controllers {
		err = c.Maintain()
		if err != nil {
			return
		}
	}
	return
}

// MaintainThorough runs the MaintainThorough method on all storages.
func MaintainThorough() (err error) {
	all := duplicateControllers()
	for _, c := range all {
		err = c.MaintainThorough()
		if err != nil {
			return
		}
	}
	return
}

// MaintainRecordStates runs record state lifecycle maintenance on all storages.
func MaintainRecordStates() error {
	all := duplicateControllers()
	now := time.Now().Unix()
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour).Unix()

	for _, c := range all {

		if c.ReadOnly() || c.Injected() {
			continue
		}

		q, err := query.New("").Check()
		if err != nil {
			return err
		}

		it, err := c.Query(q, true, true)
		if err != nil {
			return err
		}

		var toDelete []record.Record
		var toExpire []record.Record

		for r := range it.Next {
			switch {
			case r.Meta().Deleted < thirtyDaysAgo:
				toDelete = append(toDelete, r)
			case r.Meta().Expires < now:
				toExpire = append(toExpire, r)
			}
		}
		if it.Err() != nil {
			return err
		}

		for _, r := range toDelete {
			c.storage.Delete(r.DatabaseKey())
		}
		for _, r := range toExpire {
			r.Meta().Delete()
			return c.Put(r)
		}

	}
	return nil
}

func duplicateControllers() (all []*Controller) {
	controllersLock.Lock()
	defer controllersLock.Unlock()

	for _, c := range controllers {
		all = append(all, c)
	}

	return
}
