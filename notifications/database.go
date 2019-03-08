package notifications

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Safing/portbase/database"
	"github.com/Safing/portbase/database/iterator"
	"github.com/Safing/portbase/database/query"
	"github.com/Safing/portbase/database/record"
	"github.com/Safing/portbase/database/storage"
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
	notsLock.Lock()
	defer notsLock.Unlock()

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
	notsLock.Lock()
	defer notsLock.Unlock()

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
	r.Lock()
	key := r.DatabaseKey()
	n, err := EnsureNotification(r)
	r.Unlock()

	if err != nil {
		return ErrInvalidData
	}

	// transform key
	if strings.HasPrefix(key, "all/") {
		key = strings.TrimPrefix(key, "all/")
	} else {
		return ErrInvalidPath
	}

	notsLock.Lock()
	origN, ok := nots[key]
	notsLock.Unlock()

	if ok {
		n.Lock()
		defer n.Unlock()
		go origN.SelectAndExecuteAction(n.SelectedActionID)
	} else {
		// accept new notification as is
		notsLock.Lock()
		nots[key] = n
		notsLock.Unlock()
	}

	return nil
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
