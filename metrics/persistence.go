package metrics

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/safing/portbase/database"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/log"
	"github.com/tevino/abool"
)

var (
	storage       *metricsStorage
	storageLoaded = abool.New()
	storageKey    string

	db = database.NewInterface(&database.Options{
		Local:    true,
		Internal: true,
	})
)

type metricsStorage struct {
	sync.Mutex
	record.Base

	Start    time.Time
	Counters map[string]uint64
}

// EnableMetricPersistence enables metric persistence for metrics that opted
// for it. They given key is the database key where the metric data will be
// persisted.
// This call also directly loads the stored data from the database.
// The returned error is only about loading the metrics, not about enabling
// persistence.
// May only be called once.
func EnableMetricPersistence(key string) error {
	// Check if already loaded.
	if storageLoaded.IsSet() {
		return nil
	}

	// Set storage key.
	storageKey = key

	// Load metrics from storage.
	var err error
	storage, err = getMetricsStorage(key)
	switch {
	case err == nil:
		// Continue.
	case errors.Is(err, database.ErrNotFound):
		return nil
	default:
		return err
	}
	storageLoaded.Set()

	// Load saved state for all counter metrics.
	registryLock.RLock()
	defer registryLock.RUnlock()

	for _, m := range registry {
		counter, ok := m.(*Counter)
		if ok {
			counter.loadState()
		}
	}

	return nil
}

func (c *Counter) loadState() {
	// Check if we can and should load the state.
	if !storageLoaded.IsSet() || !c.Opts().Persist {
		return
	}

	c.Set(storage.Counters[c.LabeledID()])
}

func storePersistentMetrics() {
	// Check if persistence is enabled.
	if storageKey == "" {
		return
	}

	// Create new storage.
	newStorage := &metricsStorage{
		Start:    time.Now(),
		Counters: make(map[string]uint64),
	}
	newStorage.SetKey(storageKey)
	// Copy values from previous version.
	if storageLoaded.IsSet() {
		newStorage.Start = storage.Start
	}

	registryLock.RLock()
	defer registryLock.RUnlock()

	// Export all counter metrics.
	for _, m := range registry {
		if m.Opts().Persist {
			counter, ok := m.(*Counter)
			if ok {
				newStorage.Counters[m.LabeledID()] = counter.Get()
			}
		}
	}

	// Save to database.
	err := db.Put(newStorage)
	if err != nil {
		log.Warningf("metrics: failed to save metrics storage to db: %s", err)
	}
}

func getMetricsStorage(key string) (*metricsStorage, error) {
	r, err := db.Get(key)
	if err != nil {
		return nil, err
	}

	// unwrap
	if r.IsWrapped() {
		// only allocate a new struct, if we need it
		new := &metricsStorage{}
		err = record.Unwrap(r, new)
		if err != nil {
			return nil, err
		}
		return new, nil
	}

	// or adjust type
	new, ok := r.(*metricsStorage)
	if !ok {
		return nil, fmt.Errorf("record not of type *metricsStorage, but %T", r)
	}
	return new, nil
}
