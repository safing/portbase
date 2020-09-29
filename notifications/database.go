package notifications

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/safing/portbase/database"
	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/database/storage"
	"github.com/safing/portbase/log"
)

var (
	nots     = make(map[string]*Notification)
	notsLock sync.RWMutex

	dbController *database.Controller
)

// Storage interface errors
var (
	ErrInvalidData = errors.New("invalid data, must be a notification object")
	ErrInvalidPath = errors.New("invalid path")
	ErrNoDelete    = errors.New("notifications may not be deleted, they must be handled")
)

// StorageInterface provices a storage.Interface to the configuration manager.
type StorageInterface struct {
	storage.InjectBase
}

func registerAsDatabase() error {
	_, err := database.Register(&database.Database{
		Name:        "notifications",
		Description: "Notifications",
		StorageType: "injected",
	})
	if err != nil {
		return err
	}

	controller, err := database.InjectDatabase("notifications", &StorageInterface{})
	if err != nil {
		return err
	}

	dbController = controller
	return nil
}

// Get returns a database record.
func (s *StorageInterface) Get(key string) (record.Record, error) {
	notsLock.RLock()
	defer notsLock.RUnlock()

	// transform key
	if !strings.HasPrefix(key, "all/") {
		return nil, storage.ErrNotFound
	}
	key = strings.TrimPrefix(key, "all/")

	// get notification
	not, ok := nots[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return not, nil
}

// Query returns a an iterator for the supplied query.
func (s *StorageInterface) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {
	it := iterator.New()
	go s.processQuery(q, it)
	// TODO: check local and internal

	return it, nil
}

func (s *StorageInterface) processQuery(q *query.Query, it *iterator.Iterator) {
	notsLock.RLock()
	defer notsLock.RUnlock()

	// send all notifications
	for _, n := range nots {
		if n.Meta().IsDeleted() {
			continue
		}

		if q.MatchesKey(n.DatabaseKey()) && q.MatchesRecord(n) {
			select {
			case it.Next <- n:
			case <-it.Done:
				// make sure we don't leak this goroutine if the iterator get's cancelled
				return
			}
		}
	}

	it.Finish(nil)
}

// Put stores a record in the database.
func (s *StorageInterface) Put(r record.Record) (record.Record, error) {
	// record is already locked!
	key := r.DatabaseKey()
	n, err := EnsureNotification(r)

	if err != nil {
		return nil, ErrInvalidData
	}

	// transform key
	if strings.HasPrefix(key, "all/") {
		key = strings.TrimPrefix(key, "all/")
	} else {
		return nil, ErrInvalidPath
	}

	return applyUpdate(n, key)
}

func applyUpdate(n *Notification, key string) (*Notification, error) {
	// separate goroutine in order to correctly lock notsLock
	notsLock.RLock()
	existing, ok := nots[key]
	notsLock.RUnlock()

	// ignore if already deleted

	if !ok || existing.Meta().IsDeleted() {
		// this is a completely new notification
		// we pass pushUpdate==false because the storage
		// controller will push an update on put anyway.
		n.save(false)
		return n, nil
	}

	existing.Lock()
	defer existing.Unlock()

	if existing.State == Executed {
		return existing, fmt.Errorf("action already executed")
	}

	save := false

	// check if the notification has been marked as
	// "executed externally".
	if n.State == Executed {
		log.Tracef("notifications: action for %s executed externally", n.EventID)
		existing.State = Executed
		save = true

		// in case the action has been executed immediately by the
		// sender we may need to update the SelectedActionID.
		// Though, we guard the assignments with value check
		// so partial updates that only change the
		// State property do not overwrite existing values.
		if n.SelectedActionID != "" {
			existing.SelectedActionID = n.SelectedActionID
		}
	}

	if n.SelectedActionID != "" && existing.State == Active {
		log.Tracef("notifications: selected action for %s: %s", n.EventID, n.SelectedActionID)
		existing.selectAndExecuteAction(n.SelectedActionID)
		save = true
	}

	if save {
		existing.save(false)
	}

	return existing, nil
}

// Delete deletes a record from the database.
func (s *StorageInterface) Delete(key string) error {
	// transform key
	if !strings.HasPrefix(key, "all/") {
		return storage.ErrNotFound
	}
	key = strings.TrimPrefix(key, "all/")

	notsLock.Lock()
	defer notsLock.Unlock()

	n, ok := nots[key]
	if !ok {
		return storage.ErrNotFound
	}

	n.Lock()
	defer n.Unlock()

	return n.delete(true)
}

// ReadOnly returns whether the database is read only.
func (s *StorageInterface) ReadOnly() bool {
	return false
}

// EnsureNotification ensures that the given record is a Notification and returns it.
func EnsureNotification(r record.Record) (*Notification, error) {
	// unwrap
	if r.IsWrapped() {
		// only allocate a new struct, if we need it
		new := &Notification{}
		err := record.Unwrap(r, new)
		if err != nil {
			return nil, err
		}
		return new, nil
	}

	// or adjust type
	new, ok := r.(*Notification)
	if !ok {
		return nil, fmt.Errorf("record not of type *Example, but %T", r)
	}
	return new, nil
}
