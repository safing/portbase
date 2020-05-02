package database

import (
	"context"
	"time"

	"github.com/tevino/abool"

	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
)

// Maintain runs the Maintain method on all storages.
func Maintain() (err error) {
	// copy, as we might use the very long
	all := duplicateControllers()

	for _, c := range all {
		err = c.Maintain()
		if err != nil {
			return
		}
	}
	return
}

// MaintainThorough runs the MaintainThorough method on all storages.
func MaintainThorough() (err error) {
	// copy, as we might use the very long
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
func MaintainRecordStates(ctx context.Context) error { //nolint:gocognit
	// TODO: Put this in the storage interface to correctly maintain on all storages.
	// Storages might check for deletion and expiry in the query interface and not return anything here.

	// listen for ctx cancel
	stop := abool.New()
	doneCh := make(chan struct{}) // for goroutine cleanup
	defer close(doneCh)
	go func() {
		select {
		case <-ctx.Done():
		case <-doneCh:
		}
		stop.Set()
	}()

	// copy, as we might use the very long
	all := duplicateControllers()

	now := time.Now().Unix()
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour).Unix()

	for _, c := range all {
		if stop.IsSet() {
			return nil
		}

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

	queryLoop:
		for {
			select {
			case r := <-it.Next:
				if r == nil {
					break queryLoop
				}

				meta := r.Meta()
				switch {
				case meta.Deleted > 0 && meta.Deleted < thirtyDaysAgo:
					toDelete = append(toDelete, r)
				case meta.Expires > 0 && meta.Expires < now:
					toExpire = append(toExpire, r)
				}
			case <-ctx.Done():
				it.Cancel()
				break queryLoop
			}
		}
		if it.Err() != nil {
			return err
		}
		if stop.IsSet() {
			return nil
		}

		for _, r := range toDelete {
			err := c.storage.Delete(r.DatabaseKey())
			if err != nil {
				return err
			}
			if stop.IsSet() {
				return nil
			}
		}

		for _, r := range toExpire {
			r.Meta().Delete()
			err := c.Put(r)
			if err != nil {
				return err
			}
			if stop.IsSet() {
				return nil
			}
		}

	}
	return nil
}

func duplicateControllers() (all []*Controller) {
	controllersLock.RLock()
	defer controllersLock.RUnlock()

	all = make([]*Controller, 0, len(controllers))
	for _, c := range controllers {
		all = append(all, c)
	}

	return
}
