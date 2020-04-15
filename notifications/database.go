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
	dbInterface  *database.Interface

	persistentBasePath string
)

// Storage interface errors
var (
	ErrInvalidData = errors.New("invalid data, must be a notification object")
	ErrInvalidPath = errors.New("invalid path")
	ErrNoDelete    = errors.New("notifications may not be deleted, they must be handled")
)

// SetPersistenceBasePath sets the base path for persisting persistent notifications.
func SetPersistenceBasePath(dbBasePath string) {
	if persistentBasePath == "" {
		persistentBasePath = dbBasePath
	}
}

// StorageInterface provices a storage.Interface to the configuration manager.
type StorageInterface struct {
	storage.InjectBase
}

func registerAsDatabase() error {
	_, err := database.Register(&database.Database{
		Name:        "notifications",
		Description: "Notifications",
		StorageType: "injected",
		PrimaryAPI:  "",
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
	if strings.HasPrefix(key, "all/") {
		key = strings.TrimPrefix(key, "all/")
	} else {
		return nil, storage.ErrNotFound
	}

	// get notification
	not, ok := nots[key]
	if ok {
		return not, nil
	}
	return nil, storage.ErrNotFound
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
			it.Next <- n
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

	// continue in goroutine
	go UpdateNotification(n, key)

	return n, nil
}

// UpdateNotification updates a notification with input from a database action. Notification will not be saved/propagated if there is no valid change.
func UpdateNotification(n *Notification, key string) {
	n.Lock()
	defer n.Unlock()

	// separate goroutine in order to correctly lock notsLock
	notsLock.RLock()
	origN, ok := nots[key]
	notsLock.RUnlock()

	save := false

	// ignore if already deleted
	if ok && origN.Meta().IsDeleted() {
		ok = false
	}

	if ok {
		// existing notification
		// only update select attributes
		origN.Lock()
		defer origN.Unlock()
	} else {
		// new notification (from external source): old == new
		origN = n
		save = true
	}

	switch {
	case n.SelectedActionID != "" && n.Responded == 0:
		// select action, if not yet already handled
		log.Tracef("notifications: selected action for %s: %s", n.ID, n.SelectedActionID)
		origN.selectAndExecuteAction(n.SelectedActionID)
		save = true
	case origN.Executed == 0 && n.Executed != 0:
		log.Tracef("notifications: action for %s executed externally", n.ID)
		origN.Executed = n.Executed
		save = true
	}

	if save {
		// we may be locking
		go origN.Save()
	}
}

// Delete deletes a record from the database.
func (s *StorageInterface) Delete(key string) error {
	// transform key
	if strings.HasPrefix(key, "all/") {
		key = strings.TrimPrefix(key, "all/")
	} else {
		return storage.ErrNotFound
	}

	// get notification
	notsLock.Lock()
	n, ok := nots[key]
	notsLock.Unlock()
	if !ok {
		return storage.ErrNotFound
	}
	// delete
	return n.Delete()
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
