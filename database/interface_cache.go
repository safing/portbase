package database

import (
	"context"
	"errors"
	"time"

	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/log"
)

// DelayedCacheWriter must be run by the caller of an interface that uses delayed cache writing.
func (i *Interface) DelayedCacheWriter(ctx context.Context) error {
	// Check if the DelayedCacheWriter should be run at all.
	if i.options.CacheSize <= 0 || len(i.options.DelayCachedWrites) == 0 {
		return errors.New("delayed cache writer is not applicable to this database interface")
	}

	// Check if backend support the Batcher interface.
	batchPut := i.PutMany(i.options.DelayCachedWrites)
	// End batchPut immediately and check for an error.
	err := batchPut(nil)
	if err != nil {
		return err
	}

	// percentThreshold defines the minimum percentage of entries in the write cache in relation to the cache size that need to be present in order for flushing the cache to the database storage.
	percentThreshold := 25 // %
	thresholdWriteTicker := time.NewTicker(5 * time.Second)
	forceWriteTicker := time.NewTicker(5 * time.Minute)

	for {
		// Wait for trigger for writing the cache.
		select {
		case <-ctx.Done():
			// The caller is shutting down, flush the cache to storage and exit.
			i.flushWriteCache(0)
			return nil

		case <-i.triggerCacheWrite:
			// An entry from the cache was evicted that was also in the write cache.
			// This makes it likely that other entries that are also present in the
			// write cache will be evicted soon. Flush the write cache to storage
			// immediately in order to reduce single writes.
			i.flushWriteCache(0)

		case <-thresholdWriteTicker.C:
			// Often check if the the write cache has filled up to a certain degree and
			// flush it to storage before we start evicting to-be-written entries and
			// slow down the hot path again.
			i.flushWriteCache(percentThreshold)

		case <-forceWriteTicker.C:
			// Once in a while, flush the write cache to storage no matter how much
			// it is filled. We don't want entries lingering around in the write
			// cache forever. This also reduces the amount of data loss in the event
			// of a total crash.
			i.flushWriteCache(0)
		}

	}
}

func (i *Interface) flushWriteCache(percentThreshold int) {
	i.writeCacheLock.Lock()
	defer i.writeCacheLock.Unlock()

	// Check if there is anything to do.
	if len(i.writeCache) == 0 {
		return
	}

	// Check if we reach the given threshold for writing to storage.
	if (len(i.writeCache)*100)/i.options.CacheSize < percentThreshold {
		return
	}

	// Write the full cache in a batch operation.
	batchPut := i.PutMany(i.options.DelayCachedWrites)
	for _, r := range i.writeCache {
		err := batchPut(r)
		if err != nil {
			log.Warningf("database: failed to write write-cached entry to %q database: %s", i.options.DelayCachedWrites, err)
		}
	}
	// Finish batch.
	err := batchPut(nil)
	if err != nil {
		log.Warningf("database: failed to finish flushing write cache to %q database: %s", i.options.DelayCachedWrites, err)
	}

	// Optimized map clearing following the Go1.11 recommendation.
	for key := range i.writeCache {
		delete(i.writeCache, key)
	}
}

// cacheEvictHandler is run by the cache for every entry that gets evicted
// from the cache.
func (i *Interface) cacheEvictHandler(keyData, _ interface{}) {
	// Transform the key into a string.
	key, ok := keyData.(string)
	if !ok {
		return
	}

	// Check if the evicted record is one that is to be written.
	i.writeCacheLock.Lock()
	r, ok := i.writeCache[key]
	if ok {
		delete(i.writeCache, key)
	}
	i.writeCacheLock.Unlock()
	if !ok {
		return
	}

	// Write record to database in order to mitigate race conditions where the record would appear
	// as non-existent for a short duration.
	db, err := getController(r.DatabaseName())
	if err != nil {
		log.Warningf("database: failed to write evicted cache entry %q: database %q does not exist", key, r.DatabaseName())
		return
	}
	err = db.Put(r)
	if err != nil {
		log.Warningf("database: failed to write evicted cache entry %q to database: %s", key, err)
	}

	// Finally, trigger writing the full write cache because a to-be-written
	// entry was just evicted from the cache, and this makes it likely that more
	// to-be-written entries will be evicted shortly.
	select {
	case i.triggerCacheWrite <- struct{}{}:
	default:
	}
}

func (i *Interface) checkCache(key string) record.Record {
	// Check if cache is in use.
	if i.cache == nil {
		return nil
	}

	// Check if record exists in cache.
	cacheVal, err := i.cache.Get(key)
	if err == nil {
		r, ok := cacheVal.(record.Record)
		if ok {
			return r
		}
	}
	return nil
}

func (i *Interface) updateCache(r record.Record, write bool) (written bool) {
	// Check if cache is in use.
	if i.cache == nil {
		return false
	}

	// Check if record should be deleted
	if r.Meta().IsDeleted() {
		// Remove entry from cache.
		i.cache.Remove(r.Key())
		// Let write through to database storage.
		return false
	}

	// Update cache with record.
	ttl := r.Meta().GetRelativeExpiry()
	if ttl >= 0 {
		_ = i.cache.SetWithExpire(
			r.Key(),
			r,
			time.Duration(ttl)*time.Second,
		)
	} else {
		_ = i.cache.Set(
			r.Key(),
			r,
		)
	}

	// Add record to write cache instead if:
	// 1. The record is being written.
	// 2. Write delaying is active.
	// 3. Write delaying is active for the database of this record.
	if write && len(i.options.DelayCachedWrites) > 0 && r.DatabaseName() == i.options.DelayCachedWrites {
		i.writeCacheLock.Lock()
		defer i.writeCacheLock.Unlock()
		i.writeCache[r.Key()] = r
		return true
	}

	return false
}
