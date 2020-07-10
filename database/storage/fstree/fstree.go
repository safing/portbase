/*
Package fstree provides a dead simple file-based database storage backend.
It is primarily meant for easy testing or storing big files that can easily be accesses directly, without datastore.
*/
package fstree

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/safing/portbase/database/iterator"
	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/database/storage"

	"github.com/google/renameio"
)

const (
	defaultFileMode = os.FileMode(int(0644))
	defaultDirMode  = os.FileMode(int(0755))
	onWindows       = runtime.GOOS == "windows"
)

// FSTree database storage.
type FSTree struct {
	name     string
	basePath string
}

func init() {
	_ = storage.Register("fstree", NewFSTree)
}

// NewFSTree returns a (new) FSTree database.
func NewFSTree(name, location string) (storage.Interface, error) {
	basePath, err := filepath.Abs(location)
	if err != nil {
		return nil, fmt.Errorf("fstree: failed to validate path %s: %w", location, err)
	}

	file, err := os.Stat(basePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("fstree: failed to stat path %s: %w", basePath, err)
		}

		err = os.MkdirAll(basePath, defaultDirMode)
		if err != nil {
			return nil, fmt.Errorf("fstree: failed to create directory %s: %w", basePath, err)
		}
	} else if !file.IsDir() {
		return nil, fmt.Errorf("fstree: provided database path (%s) is a file", basePath)
	}

	return &FSTree{
		name:     name,
		basePath: basePath,
	}, nil
}

func (fst *FSTree) buildFilePath(key string, checkKeyLength bool) (string, error) {
	// check key length
	if checkKeyLength && len(key) < 1 {
		return "", fmt.Errorf("fstree: key too short: %s", key)
	}
	// build filepath
	dstPath := filepath.Join(fst.basePath, key) // Join also calls Clean()
	if !strings.HasPrefix(dstPath, fst.basePath) {
		return "", fmt.Errorf("fstree: key integrity check failed, compiled path is %s", dstPath)
	}
	// return
	return dstPath, nil
}

// Get returns a database record.
func (fst *FSTree) Get(key string) (record.Record, error) {
	dstPath, err := fst.buildFilePath(key, true)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(dstPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("fstree: failed to read file %s: %s", dstPath, err)
	}

	r, err := record.NewRawWrapper(fst.name, key, data)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// Put stores a record in the database.
func (fst *FSTree) Put(r record.Record) (record.Record, error) {
	dstPath, err := fst.buildFilePath(r.DatabaseKey(), true)
	if err != nil {
		return nil, err
	}

	data, err := r.MarshalRecord(r)
	if err != nil {
		return nil, err
	}

	err = writeFile(dstPath, data, defaultFileMode)
	if err != nil {
		// create dir and try again
		err = os.MkdirAll(filepath.Dir(dstPath), defaultDirMode)
		if err != nil {
			return nil, fmt.Errorf("fstree: failed to create directory %s: %s", filepath.Dir(dstPath), err)
		}
		err = writeFile(dstPath, data, defaultFileMode)
		if err != nil {
			return nil, fmt.Errorf("fstree: could not write file %s: %s", dstPath, err)
		}
	}

	return r, nil
}

// Delete deletes a record from the database.
func (fst *FSTree) Delete(key string) error {
	dstPath, err := fst.buildFilePath(key, true)
	if err != nil {
		return err
	}

	// remove entry
	err = os.Remove(dstPath)
	if err != nil {
		return fmt.Errorf("fstree: could not delete %s: %s", dstPath, err)
	}

	return nil
}

// Query returns a an iterator for the supplied query.
func (fst *FSTree) Query(q *query.Query, local, internal bool) (*iterator.Iterator, error) {
	_, err := q.Check()
	if err != nil {
		return nil, fmt.Errorf("invalid query: %s", err)
	}

	walkPrefix, err := fst.buildFilePath(q.DatabaseKeyPrefix(), false)
	if err != nil {
		return nil, err
	}
	fileInfo, err := os.Stat(walkPrefix)
	var walkRoot string
	switch {
	case err == nil && fileInfo.IsDir():
		walkRoot = walkPrefix
	case err == nil:
		walkRoot = filepath.Dir(walkPrefix)
	case os.IsNotExist(err):
		walkRoot = filepath.Dir(walkPrefix)
	default: // err != nil
		return nil, fmt.Errorf("fstree: could not stat query root %s: %s", walkPrefix, err)
	}

	queryIter := iterator.New()

	go fst.queryExecutor(walkRoot, queryIter, q, local, internal)
	return queryIter, nil
}

func (fst *FSTree) queryExecutor(walkRoot string, queryIter *iterator.Iterator, q *query.Query, local, internal bool) {
	err := filepath.Walk(walkRoot, func(path string, info os.FileInfo, err error) error {

		// check for error
		if err != nil {
			return fmt.Errorf("fstree: error in walking fs: %s", err)
		}

		if info.IsDir() {
			// skip dir if not in scope
			if !strings.HasPrefix(path, fst.basePath) {
				return filepath.SkipDir
			}
			// continue
			return nil
		}

		// still in scope?
		if !strings.HasPrefix(path, fst.basePath) {
			return nil
		}

		// read file
		data, err := ioutil.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("fstree: failed to read file %s: %s", path, err)
		}

		// parse
		key, err := filepath.Rel(fst.basePath, path)
		if err != nil {
			return fmt.Errorf("fstree: failed to extract key from filepath %s: %s", path, err)
		}
		r, err := record.NewRawWrapper(fst.name, key, data)
		if err != nil {
			return fmt.Errorf("fstree: failed to load file %s: %s", path, err)
		}

		if !r.Meta().CheckValidity() {
			// record is not valid
			return nil
		}

		if !r.Meta().CheckPermission(local, internal) {
			// no permission to access
			return nil
		}

		// check if matches, then send
		if q.MatchesRecord(r) {
			select {
			case queryIter.Next <- r:
			case <-queryIter.Done:
			case <-time.After(1 * time.Second):
				return errors.New("fstree: query buffer full, timeout")
			}
		}

		return nil
	})

	queryIter.Finish(err)
}

// ReadOnly returns whether the database is read only.
func (fst *FSTree) ReadOnly() bool {
	return false
}

// Injected returns whether the database is injected.
func (fst *FSTree) Injected() bool {
	return false
}

// Maintain runs a light maintenance operation on the database.
func (fst *FSTree) Maintain(_ context.Context) error {
	return nil
}

// MaintainThorough runs a thorough maintenance operation on the database.
func (fst *FSTree) MaintainThorough(_ context.Context) error {
	return nil
}

// MaintainRecordStates maintains records states in the database.
func (fst *FSTree) MaintainRecordStates(ctx context.Context, purgeDeletedBefore time.Time) error {
	// TODO: implement MaintainRecordStates
	return nil
}

// Shutdown shuts down the database.
func (fst *FSTree) Shutdown() error {
	return nil
}

// writeFile mirrors ioutil.WriteFile, replacing an existing file with the same
// name atomically. This is not atomic on Windows, but still an improvement.
// TODO: Replace with github.com/google/renamio.WriteFile as soon as it is fixed on Windows.
// This function is forked from https://github.com/google/renameio/blob/a368f9987532a68a3d676566141654a81aa8100b/writefile.go.
func writeFile(filename string, data []byte, perm os.FileMode) error {
	t, err := renameio.TempFile("", filename)
	if err != nil {
		return err
	}
	defer t.Cleanup() //nolint:errcheck

	// Set permissions before writing data, in case the data is sensitive.
	if !onWindows {
		if err := t.Chmod(perm); err != nil {
			return err
		}
	}

	if _, err := t.Write(data); err != nil {
		return err
	}

	return t.CloseAtomicallyReplace()
}
