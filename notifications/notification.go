package notifications

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/log"
	"github.com/safing/portbase/utils"
)

// Type describes the type of a notification.
type Type uint8

// Notification types.
const (
	Info    Type = 0
	Warning Type = 1
	Prompt  Type = 2
)

// State describes the state of a notification.
type State string

// NotificationActionFn defines the function signature for notification action
// functions.
type NotificationActionFn func(context.Context, *Notification) error

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
	actionFunction NotificationActionFn // call function to process action
	actionTrigger  chan string          // and/or send to a channel
	expiredTrigger chan struct{}        // closed on expire
}

// Action describes an action that can be taken for a notification.
type Action struct {
	ID   string
	Text string
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
	return n.save(true)
}

// save saves the notification to the internal storage. It locks the
// notification, so it must not be locked when save is called.
func (n *Notification) save(pushUpdate bool) *Notification {
	var id string

	// Delete notification after processing deletion.
	defer func() {
		// Lock and save to notification storage.
		notsLock.Lock()
		defer notsLock.Unlock()
		nots[id] = n
	}()

	// We do not access EventData here, so it is enough to just lock the
	// notification itself.
	n.lock.Lock()
	defer n.lock.Unlock()

	// Save ID for deletion
	id = n.EventID

	// Generate random GUID if not set.
	if n.GUID == "" {
		n.GUID = utils.RandomUUID(n.EventID).String()
	}

	// Make ack notification if there are no defined actions.
	if len(n.AvailableActions) == 0 {
		n.AvailableActions = []*Action{
			{
				ID:   "ack",
				Text: "OK",
			},
		}
	}

	// Make sure we always have a notification state assigned.
	if n.State == "" {
		n.State = Active
	}

	// check key
	if !n.KeyIsSet() {
		n.SetKey(fmt.Sprintf("notifications:all/%s", n.EventID))
	}

	// Update meta data.
	n.UpdateMeta()

	// Push update via the database system if needed.
	if pushUpdate {
		log.Tracef("notifications: pushing update for %s to subscribers", n.Key())
		dbController.PushUpdate(n)
	}

	return n
}

// SetActionFunction sets a trigger function to be executed when the user reacted on the notification.
// The provided function will be started as its own goroutine and will have to lock everything it accesses, even the provided notification.
func (n *Notification) SetActionFunction(fn NotificationActionFn) *Notification {
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
	// Save when we're finished, if needed.
	save := false
	defer func() {
		if save {
			n.save(true)
		}
	}()

	n.lock.Lock()
	defer n.lock.Unlock()

	// Don't update if notification isn't active.
	if n.State != Active {
		return
	}

	// Don't update too quickly.
	if n.Meta().Modified > time.Now().Add(-10*time.Second).Unix() {
		return
	}

	// Update expiry and save.
	n.Expires = expires
	save = true
}

// Delete (prematurely) cancels and deletes a notification.
func (n *Notification) Delete() error {
	n.delete(true)
	return nil
}

// delete deletes the notification from the internal storage. It locks the
// notification, so it must not be locked when delete is called.
func (n *Notification) delete(pushUpdate bool) {
	var id string

	// Delete notification after processing deletion.
	defer func() {
		// Lock and delete from notification storage.
		notsLock.Lock()
		defer notsLock.Unlock()
		delete(nots, id)
	}()

	// We do not access EventData here, so it is enough to just lock the
	// notification itself.
	n.lock.Lock()
	defer n.lock.Unlock()

	// Save ID for deletion
	id = n.EventID

	// Mark notification as deleted.
	n.Meta().Delete()

	// Close expiry channel if available.
	if n.expiredTrigger != nil {
		close(n.expiredTrigger)
		n.expiredTrigger = nil
	}

	// Push update via the database system if needed.
	if pushUpdate {
		dbController.PushUpdate(n)
	}
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
		module.StartWorker("notification action execution", func(ctx context.Context) error {
			return n.actionFunction(ctx, n)
		})
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
