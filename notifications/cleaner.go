package notifications

import (
	"context"
	"time"

	"github.com/safing/portbase/log"
)

//nolint:unparam // must conform to interface
func cleaner(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
			cleanNotifications()
		}
	}
}

func cleanNotifications() {
	now := time.Now().Unix()
	finishedThreshhold := time.Now().Add(-10 * time.Second).Unix()
	executionTimelimit := time.Now().Add(-24 * time.Hour).Unix()
	fallbackTimelimit := time.Now().Add(-72 * time.Hour).Unix()

	notsLock.Lock()
	defer notsLock.Unlock()

	for _, n := range nots {
		n.Lock()
		switch {
		case n.Executed != 0: // notification was fully handled
			// wait for a short time before deleting
			if n.Executed < finishedThreshhold {
				go deleteNotification(n)
			}
		case n.Responded != 0:
			// waiting for execution
			if n.Responded < executionTimelimit {
				go deleteNotification(n)
			}
		case n.Expires != 0:
			// expired without response
			if n.Expires < now {
				go deleteNotification(n)
			}
		case n.Created != 0:
			// fallback: delete after 3 days after creation
			if n.Created < fallbackTimelimit {
				go deleteNotification(n)
			}
		default:
			// invalid, impossible to determine cleanup timeframe, delete now
			go deleteNotification(n)
		}
		n.Unlock()
	}
}

func deleteNotification(n *Notification) {
	err := n.Delete()
	if err != nil {
		log.Debugf("notifications: failed to delete %s: %s", n.ID, err)
	}
}
