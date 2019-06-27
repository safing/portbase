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
		if q.MatchesKey(n.DatabaseKey()) && q.MatchesRecord(n) {
			it.Next <- n
		}
	}

	it.Finish(nil)
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

// Put stores a record in the database.
func (s *StorageInterface) Put(r record.Record) error {
	// record is already locked!
	key := r.DatabaseKey()
	n, err := EnsureNotification(r)

	if err != nil {
		return ErrInvalidData
	}

	// transform key
	if strings.HasPrefix(key, "all/") {
		key = strings.TrimPrefix(key, "all/")
	} else {
		return ErrInvalidPath
	}

	// continue in goroutine
	go updateNotificationFromDatabasePut(n, key)

	return nil
}

func updateNotificationFromDatabasePut(n *Notification, key string) {
	// seperate goroutine in order to correctly lock notsLock
	notsLock.RLock()
	origN, ok := nots[key]
	notsLock.RUnlock()

	if ok {
		// existing notification, update selected action ID only
		n.Lock()
		defer n.Unlock()
		if n.SelectedActionID != "" {
			log.Tracef("notifications: user selected action for %s: %s", n.ID, n.SelectedActionID)
			go origN.SelectAndExecuteAction(n.SelectedActionID)
		}
	} else {
		// accept new notification as is
		notsLock.Lock()
		nots[key] = n
		notsLock.Unlock()
	}
}

// Delete deletes a record from the database.
func (s *StorageInterface) Delete(key string) error {
	return ErrNoDelete
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
