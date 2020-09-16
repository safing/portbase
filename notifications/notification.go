package notifications

import (
	"fmt"
	"sync"
	"time"

	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/log"
	"github.com/safing/portbase/utils"
)

// Type describes the type of a notification.
type Type uint8

// Notification types
const (
	Info    Type = 0
	Warning Type = 1
	Prompt  Type = 2
)

// State describes the state of a notification.
type State string

// Possible notification states.
// State transitions can only happen from top to bottom.
const (
	// Active describes a notification that is active, no expired and,
	// if actions are available, still waits for the user to select an
	// action.
	Active State = "active"
	// Responded describes a notification where the user has already
	// selected which action to take but that action is still to be
	// performed.
	Responded State = "responded"
	// Executes describes a notification where the user has selected
	// and action and that action has been performed.
	Executed State = "executed"
)

// Notification represents a notification that is to be delivered to the user.
type Notification struct {
	record.Base
	// EventID is used to identify a specific notification. It consists of
	// the module name and a per-module unique event id.
	// The following format is recommended:
	// 	<module-id>:<event-id>
	EventID string
	// GUID is a unique identifier for each notification instance. That is
	// two notifications with the same EventID must still have unique GUIDs.
	// The GUID is mainly used for system (Windows) integration and is
	// automatically populated by the notification package. Average users
	// don't need to care about this field.
	GUID string
	// Type is the notification type. It can be one of Info, Warning or Prompt.
	Type Type
	// Message is the default message shown to the user if no localized version
	// of the notification is available. Note that the message should already
	// have any paramerized values replaced.
	Message string
	// EventData contains an additional payload for the notification. This payload
	// may contain contextual data and may be used by a localization framework
	// to populate the notification message template.
	// If EventData implements sync.Locker it will be locked and unlocked together with the
	// notification. Otherwise, EventData is expected to be immutable once the
	// notification has been saved and handed over to the notification or database package.
	EventData interface{}
	// Expires holds the unix epoch timestamp at which the notification expires
	// and can be cleaned up.
	// Users can safely ignore expired notifications and should handle expiry the
	// same as deletion.
	Expires int64
	// State describes the current state of a notification. See State for
	// a list of available values and their meaning.
	State State
	// AvailableActions defines a list of actions that a user can choose from.
	AvailableActions []*Action
	// SelectedActionID is updated to match the ID of one of the AvailableActions
	// based on the user selection.
	SelectedActionID string

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

func noOpAction(n *Notification) {}

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

// NotifyInfo is a helper method for quickly showing a info
// notification. The notification is already shown. If id is
// an empty string a new UUIDv4 will be generated.
func NotifyInfo(id, msg string, actions ...Action) *Notification {
	return notify(Info, id, msg, actions...)
}

// NotifyWarn is a helper method for quickly showing a warning
// notification. The notification is already shown. If id is
// an empty string a new UUIDv4 will be generated.
func NotifyWarn(id, msg string, actions ...Action) *Notification {
	return notify(Warning, id, msg, actions...)
}

// NotifyPrompt is a helper method for quickly showing a prompt
// notification. The notification is already shown. If id is
// an empty string a new UUIDv4 will be generated.
func NotifyPrompt(id, msg string, actions ...Action) *Notification {
	return notify(Prompt, id, msg, actions...)
}

func notify(nType Type, id string, msg string, actions ...Action) *Notification {
	acts := make([]*Action, len(actions))
	for idx := range actions {
		a := actions[idx]
		acts[idx] = &a
	}

	if id == "" {
		id = utils.DerivedInstanceUUID(msg).String()
	}

	n := Notification{
		EventID:          id,
		Message:          msg,
		Type:             nType,
		AvailableActions: acts,
	}

	return n.Save()
}

// Save saves the notification and returns it.
func (n *Notification) Save() *Notification {
	n.Lock()
	defer n.Unlock()

	return n.save(true)
}

func (n *Notification) save(pushUpdate bool) *Notification {
	if n.GUID == "" {
		n.GUID = utils.RandomUUID(n.EventID).String()
	}

	// make ack notification if there are no defined actions
	if len(n.AvailableActions) == 0 {
		n.AvailableActions = []*Action{
			{
				ID:   "ack",
				Text: "OK",
			},
		}
		n.actionFunction = noOpAction
	}

	// Make sure we always have a reasonable expiration set.
	if n.Expires == 0 {
		n.Expires = time.Now().Add(72 * time.Hour).Unix()
	}

	// check key
	if n.DatabaseKey() == "" {
		n.SetKey(fmt.Sprintf("notifications:all/%s", n.EventID))
	}

	n.UpdateMeta()

	// store the notification inside or map
	notsLock.Lock()
	nots[n.EventID] = n
	notsLock.Unlock()

	if pushUpdate {
		log.Tracef("notifications: pushing update for %s to subscribers", n.Key())
		dbController.PushUpdate(n)
	}

	return n
}

// SetActionFunction sets a trigger function to be executed when the user reacted on the notification.
// The provided function will be started as its own goroutine and will have to lock everything it accesses, even the provided notification.
func (n *Notification) SetActionFunction(fn func(*Notification)) *Notification {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.actionFunction = fn
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

// Update updates/resends a notification if it was not already responded to.
func (n *Notification) Update(expires int64) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.State == Active {
		n.Expires = expires
		n.save(true)
	}
}

// Delete (prematurely) cancels and deletes a notification.
func (n *Notification) Delete() error {
	notsLock.Lock()
	defer notsLock.Unlock()
	n.Lock()
	defer n.Unlock()

	return n.delete(true)
}

func (n *Notification) delete(pushUpdate bool) error {
	// mark as deleted
	n.Meta().Delete()

	// delete from internal storage
	delete(nots, n.EventID)

	// close expired
	if n.expiredTrigger != nil {
		close(n.expiredTrigger)
		n.expiredTrigger = nil
	}

	if pushUpdate {
		dbController.PushUpdate(n)
	}

	return nil
}

// Expired notifies the caller when the notification has expired.
func (n *Notification) Expired() <-chan struct{} {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.expiredTrigger == nil {
		n.expiredTrigger = make(chan struct{})
	}

	return n.expiredTrigger
}

// selectAndExecuteAction sets the user response and executes/triggers the action, if possible.
func (n *Notification) selectAndExecuteAction(id string) {
	if n.State != Active {
		return
	}

	n.State = Responded
	n.SelectedActionID = id

	executed := false
	if n.actionFunction != nil {
		go n.actionFunction(n)
		executed = true
	}

	if n.actionTrigger != nil {
		// satisfy all listeners (if they are listening)
		// TODO(ppacher): if we miss to notify the waiter here (because
		//                nobody is listeing on actionTrigger) we wil likely
		//                never be able to execute the action again (simply because
		//                we won't try). May consider replacing the single actionTrigger
		//                channel with a per-listener (buffered) one so we just send
		//                the value and close the channel.
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

	if executed {
		n.State = Executed
	}
}

// Lock locks the Notification. If EventData is set and
// implements sync.Locker it is locked as well. Users that
// want to replace the EventData on a notification must
// ensure to unlock the current value on their own. If the
// new EventData implements sync.Locker as well, it must
// be locked prior to unlocking the notification.
func (n *Notification) Lock() {
	n.lock.Lock()
	if locker, ok := n.EventData.(sync.Locker); ok {
		locker.Lock()
	}
}

// Unlock unlocks the Notification and the EventData, if
// it implements sync.Locker. See Lock() for more information
// on how to replace and work with EventData.
func (n *Notification) Unlock() {
	n.lock.Unlock()
	if locker, ok := n.EventData.(sync.Locker); ok {
		locker.Unlock()
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
