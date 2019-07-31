package notifications

import (
	"fmt"
	"sync"
	"time"

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
		duplicate.SetKey(fmt.Sprintf("%s/%s", persistentBasePath, n.ID))
		go func() {
			err := dbInterface.Put(duplicate)
			if err != nil {
				log.Warningf("notifications: failed to persist notification %s: %s", n.Key(), err)
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
	defer n.lock.Unlock()

	if n.actionTrigger == nil {
		n.actionTrigger = make(chan string)
	}

	return n.actionTrigger
}

// Cancel (prematurely) destroys a notification.
func (n *Notification) Cancel() {
	notsLock.Lock()
	defer notsLock.Unlock()
	n.Lock()
	defer n.Unlock()

	// delete
	n.Meta().Delete()
	delete(nots, n.ID)

	// save (ie. propagate delete)
	go n.Save()
}

// SelectAndExecuteAction sets the user response and executes/triggers the action, if possible.
func (n *Notification) SelectAndExecuteAction(id string) {
	n.Lock()
	defer n.Unlock()

	// update selection
	if n.Executed != 0 {
		// we already executed
		return
	}
	n.SelectedActionID = id
	n.Responded = time.Now().Unix()

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
			default:
				break triggerAll
			}
		}
	}

	// save execution time
	if executed {
		n.Executed = time.Now().Unix()
	}

	go n.Save()
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
