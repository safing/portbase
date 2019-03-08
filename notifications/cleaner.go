package notifications

import (
	"time"
)

func cleaner() {
	shutdownWg.Add(1)
	select {
	case <-shutdownSignal:
		shutdownWg.Done()
		return
	case <-time.After(1 * time.Minute):
		cleanNotifications()
	}
}

func cleanNotifications() {
	threshold := time.Now().Add(-2 * time.Minute).Unix()
	maxThreshold := time.Now().Add(-72 * time.Hour).Unix()

	notsLock.Lock()
	defer notsLock.Unlock()

	for _, n := range nots {
		n.Lock()
		if n.Expires != 0 && n.Expires < threshold ||
			n.Executed != 0 && n.Executed < threshold ||
			n.Created < maxThreshold {

			// delete
			n.Meta().Delete()
			delete(nots, n.ID)

			// save (ie. propagate delete)
			go n.Save()
		}
		n.Unlock()
	}
}
