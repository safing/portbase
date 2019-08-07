package notifications

import (
	"fmt"
	"sync"
	"time"

	"github.com/safing/portbase/database"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/log"

	uuid "github.com/satori/go.uuid"
)

// Notification types
const (
	Info    uint8 = 0
	Warning uint8 = 1
	Prompt  uint8 = 2
)

// Notification represents a notification that is to be delivered to the user.
type Notification struct {
	record.Base

	ID   string
	GUID string

	Message string
	// MessageTemplate string
	// MessageData []string
	DataSubject sync.Locker
	Type        uint8

	AvailableActions []*Action
	SelectedActionID string

	Persistent bool  // this notification persists until it is handled and survives restarts
	Created    int64 // creation timestamp, notification "starts"
	Expires    int64 // expiry timestamp, notification is expected to be canceled at this time and may be cleaned up afterwards
	Responded  int64 // response timestamp, notification "ends"
	Executed   int64 // execution timestamp, notification will be deleted soon

	lock           sync.Mutex
	actionFunction func(*Notification) // call function to process action
	actionTrigger  chan string         // and/or send to a channel
	expiredTrigger chan struct{}       // closed on expire
}

// Action describes an action that can be taken for a notification.
type Action struct {
	ID   string
	Text string
}

func noOpAction(n *Notification) {
	return
}

// Get returns the notification identifed by the given id or nil if it doesn't exist.
func Get(id string) *Notification {
	notsLock.RLock()
	defer notsLock.RUnlock()
	n, ok := nots[id]
	if ok {
		return n
	}
	return nil
}

// Save saves the notification and returns it.
func (n *Notification) Save() *Notification {
	notsLock.Lock()
	defer notsLock.Unlock()
	n.Lock()
	defer n.Unlock()

	// initialize
	if n.Created == 0 {
		n.Created = time.Now().Unix()
	}
	if n.GUID == "" {
		n.GUID = uuid.NewV4().String()
	}
	// check key
	if n.DatabaseKey() == "" {
		n.SetKey(fmt.Sprintf("notifications:all/%s", n.ID))
	}

	// update meta
	n.UpdateMeta()

	// assign to data map
	nots[n.ID] = n

	// push update
	dbController.PushUpdate(n)

	// persist
	if n.Persistent && persistentBasePath != "" {
		duplicate := &Notification{
			ID:               n.ID,
			Message:          n.Message,
			DataSubject:      n.DataSubject,
			AvailableActions: duplicateActions(n.AvailableActions),
			SelectedActionID: n.SelectedActionID,
			Persistent:       n.Persistent,
			Created:          n.Created,
			Expires:          n.Expires,
			Responded:        n.Responded,
			Executed:         n.Executed,
		}
		duplicate.SetMeta(n.Meta().Duplicate())
		key := fmt.Sprintf("%s/%s", persistentBasePath, n.ID)
		duplicate.SetKey(key)
		go func() {
			err := dbInterface.Put(duplicate)
			if err != nil {
				log.Warningf("notifications: failed to persist notification %s: %s", key, err)
			}
		}()
	}

	return n
}

// SetActionFunction sets a trigger function to be executed when the user reacted on the notification.
// The provided funtion will be started as its own goroutine and will have to lock everything it accesses, even the provided notification.
func (n *Notification) SetActionFunction(fn func(*Notification)) *Notification {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.actionFunction = fn
	return n
}

// MakeAck sets a default "OK" action and a no-op action function.
func (n *Notification) MakeAck() *Notification {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.AvailableActions = []*Action{
		&Action{
			ID:   "ack",
			Text: "OK",
		},
	}
	n.Type = Info
	n.actionFunction = noOpAction

	return n
}

// Response waits for the user to respond to the notification and returns the selected action.
func (n *Notification) Response() <-chan string {
	n.lock.Lock()
	if n.actionTrigger == nil {
		n.actionTrigger = make(chan string)
	}
	n.lock.Unlock()

	return n.actionTrigger
}

// Update updates/resends a notification if it was not already responded to.
func (n *Notification) Update(expires int64) {
	responded := true
	n.lock.Lock()
	if n.Responded == 0 {
		responded = false
		n.Expires = expires
	}
	n.lock.Unlock()

	// save if not yet responded
	if !responded {
		n.Save()
	}
}

// Delete (prematurely) cancels and deletes a notification.
func (n *Notification) Delete() error {
	notsLock.Lock()
	defer notsLock.Unlock()
	n.Lock()
	defer n.Unlock()

	// mark as deleted
	n.Meta().Delete()

	// delete from internal storage
	delete(nots, n.ID)

	// close expired
	if n.expiredTrigger != nil {
		close(n.expiredTrigger)
		n.expiredTrigger = nil
	}

	// push update
	dbController.PushUpdate(n)

	// delete from persistent storage
	if n.Persistent && persistentBasePath != "" {
		key := fmt.Sprintf("%s/%s", persistentBasePath, n.ID)
		err := dbInterface.Delete(key)
		if err != nil && err != database.ErrNotFound {
			return fmt.Errorf("failed to delete persisted notification %s from database: %s", key, err)
		}
	}

	return nil
}

// Expired notifies the caller when the notification has expired.
func (n *Notification) Expired() <-chan struct{} {
	n.lock.Lock()
	if n.expiredTrigger == nil {
		n.expiredTrigger = make(chan struct{})
	}
	n.lock.Unlock()

	return n.expiredTrigger
}

// selectAndExecuteAction sets the user response and executes/triggers the action, if possible.
func (n *Notification) selectAndExecuteAction(id string) {
	// abort if already executed
	if n.Executed != 0 {
		return
	}

	// set response
	n.Responded = time.Now().Unix()
	n.SelectedActionID = id

	// execute
	executed := false
	if n.actionFunction != nil {
		go n.actionFunction(n)
		executed = true
	}
	if n.actionTrigger != nil {
		// satisfy all listeners
	triggerAll:
		for {
			select {
			case n.actionTrigger <- n.SelectedActionID:
				executed = true
			case <-time.After(100 * time.Millisecond): // mitigate race conditions
				break triggerAll
			}
		}
	}

	// save execution time
	if executed {
		n.Executed = time.Now().Unix()
	}
}

// AddDataSubject adds the data subject to the notification. This is the only way how a data subject should be added - it avoids locking problems.
func (n *Notification) AddDataSubject(ds sync.Locker) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.DataSubject = ds
}

// Lock locks the Notification and the DataSubject, if available.
func (n *Notification) Lock() {
	n.lock.Lock()
	if n.DataSubject != nil {
		n.DataSubject.Lock()
	}
}

// Unlock unlocks the Notification and the DataSubject, if available.
func (n *Notification) Unlock() {
	n.lock.Unlock()
	if n.DataSubject != nil {
		n.DataSubject.Unlock()
	}
}

func duplicateActions(original []*Action) (duplicate []*Action) {
	duplicate = make([]*Action, len(original))
	for _, action := range original {
		duplicate = append(duplicate, &Action{
			ID:   action.ID,
			Text: action.Text,
		})
	}
	return
}
