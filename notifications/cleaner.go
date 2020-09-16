package notifications

import (
	"context"
	"time"

	"github.com/safing/portbase/log"
)

func cleaner(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

L:
	for {
		select {
		case <-ctx.Done():
			break L
		case <-ticker.C:
			deleteExpiredNotifs()
		}
	}
	return nil
}

func deleteExpiredNotifs() {
	now := time.Now().Unix()

	notsLock.Lock()
	defer notsLock.Unlock()

	toDelete := make([]*Notification, 0, len(nots))
	for _, n := range nots {
		n.Lock()
		if now > n.Expires {
			toDelete = append(toDelete, n)
		}
		n.Unlock()
	}

	for _, n := range toDelete {
		n.Lock()
		err := n.delete(true)
		n.Unlock()

		if err != nil {
			log.Debugf("notifications: failed to delete %s: %s", n.EventID, err)
		}
	}
}
