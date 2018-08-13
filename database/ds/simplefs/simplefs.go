// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

/*
Package simplefs provides a dead simple file-based datastore backed.
It is primarily meant for easy testing or storing big files that can easily be accesses directly, without datastore.

  /path/path/type:key.json
  /path/path/type:key/type:key

*/
package simplefs

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	"github.com/tevino/abool"

	"github.com/Safing/safing-core/database/dbutils"
	"github.com/Safing/safing-core/formats/dsd"
	"github.com/Safing/safing-core/log"
)

const (
	SIMPLEFS_TAG     = "330adcf3924003a59ae93289bc2cbb236391588f"
	DEFAULT_FILEMODE = os.FileMode(int(0644))
	DEFAULT_DIRMODE  = os.FileMode(int(0755))
)

type datastore struct {
	basePath    string
	basePathLen int
}

func NewDatastore(path string) (*datastore, error) {
	basePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to validate path %s: %s", path, err)
	}
	tagfile := filepath.Join(basePath, ".simplefs")

	file, err := os.Stat(basePath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(basePath, DEFAULT_DIRMODE)
			if err != nil {
				return nil, fmt.Errorf("failed to create directory: %s", err)
			}
			err = ioutil.WriteFile(tagfile, []byte(SIMPLEFS_TAG), DEFAULT_FILEMODE)
			if err != nil {
				return nil, fmt.Errorf("failed to create tag file (%s): %s", tagfile, err)
			}
		} else {
			return nil, fmt.Errorf("failed to stat path: %s", err)
		}
	} else {
		if !file.IsDir() {
			return nil, fmt.Errorf("provided path (%s) is a file", basePath)
		}
		// check if valid simplefs storage dir
		content, err := ioutil.ReadFile(tagfile)
		if err != nil {
			return nil, fmt.Errorf("could not read tag file (%s): %s", tagfile, err)
		}
		if string(content) != SIMPLEFS_TAG {
			return nil, fmt.Errorf("invalid tag file (%s)", tagfile)
		}
	}

	log.Infof("simplefs: opened database at %s", basePath)
	return &datastore{
		basePath:    basePath,
		basePathLen: len(basePath),
	}, nil
}

func (d *datastore) buildPath(path string) (string, error) {
	if len(path) < 2 {
		return "", fmt.Errorf("key too short: %s", path)
	}
	newPath := filepath.Clean(filepath.Join(d.basePath, path[1:])) + ".dsd"
	if !strings.HasPrefix(newPath, d.basePath) {
		return "", fmt.Errorf("key integrity check failed, compiled path is %s", newPath)
	}
	return newPath, nil
}

func (d *datastore) Put(key ds.Key, value interface{}) (err error) {
	objPath, err := d.buildPath(key.String())
	if err != nil {
		return err
	}

	bytes, err := dbutils.DumpModel(value, dsd.AUTO) // or other dsd format
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(objPath, bytes, DEFAULT_FILEMODE)
	if err != nil {
		// create dir and try again
		err = os.MkdirAll(filepath.Dir(objPath), DEFAULT_DIRMODE)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %s", filepath.Dir(objPath), err)
		}
		err = ioutil.WriteFile(objPath, bytes, DEFAULT_FILEMODE)
		if err != nil {
			return fmt.Errorf("could not write file %s: %s", objPath, err)
		}
	}

	return nil
}

func (d *datastore) Get(key ds.Key) (interface{}, error) {
	objPath, err := d.buildPath(key.String())
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(objPath)
	if err != nil {
		// TODO: distinguish between error and inexistance
		return nil, ds.ErrNotFound
	}

	model, err := dbutils.NewWrapper(&key, data)
	if err != nil {
		return nil, err
	}

	return model, nil
}

func (d *datastore) Has(key ds.Key) (exists bool, err error) {
	objPath, err := d.buildPath(key.String())
	if err != nil {
		return false, err
	}

	_, err = os.Stat(objPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat path %s: %s", objPath, err)
	}
	return true, nil
}

func (d *datastore) Delete(key ds.Key) (err error) {
	objPath, err := d.buildPath(key.String())
	if err != nil {
		return err
	}

	// remove entry
	err = os.Remove(objPath)
	if err != nil {
		return fmt.Errorf("could not delete (all) in path %s: %s", objPath, err)
	}
	// remove children
	err = os.RemoveAll(objPath[:len(objPath)-4])
	if err != nil {
		return fmt.Errorf("could not delete (all) in path %s: %s", objPath, err)
	}

	return nil
}

func (d *datastore) Query(q dsq.Query) (dsq.Results, error) {
	if len(q.Orders) > 0 {
		return nil, fmt.Errorf("simplefs: no support for ordering queries yet")
	}

	// log.Tracef("new query: %s", q.Prefix)

	// log.Tracef("simplefs: new query with prefix %s", q.Prefix)

	walkPath, err := d.buildPath(q.Prefix)
	if err != nil {
		return nil, err
	}
	walkPath = walkPath[:strings.LastIndex(walkPath, string(os.PathSeparator))]

	files := make(chan *dsq.Entry)
	stopWalkingFlag := abool.NewBool(false)
	stopWalking := make(chan interface{})
	counter := 0

	go func() {

		err := filepath.Walk(walkPath, func(path string, info os.FileInfo, err error) error {

			// log.Tracef("walking: %s", path)

			if err != nil {
				return fmt.Errorf("simplfs: error in query: %s", err)
			}

			// skip directories
			if info.IsDir() {
				return nil
			}

			// check if we are still were we should be
			if !strings.HasPrefix(path, d.basePath) {
				log.Criticalf("simplfs: query jailbreaked: %s", path)
				return errors.New("jailbreaked")
			}
			path = path[d.basePathLen:]

			// check if there is enough space to remove ".dsd"
			if len(path) < 6 {
				return nil
			}
			path = path[:len(path)-4]

			// check if we still match prefix
			if !strings.HasPrefix(path, q.Prefix) {
				return nil
			}

			entry := dsq.Entry{
				Key: path,
				// TODO: phew, do we really want to load every single file we might not need? use nil for now.
				Value: nil,
			}
			for _, filter := range q.Filters {
				if !filter.Filter(entry) {
					return nil
				}
			}

			// yay, entry matches!
			counter++

			if q.Offset > counter {
				return nil
			}

			select {
			case files <- &entry:
			case <-stopWalking:
				return errors.New("finished")
			}

			if q.Limit != 0 && q.Limit <= counter {
				return errors.New("finished")
			}

			return nil

		})

		if err != nil {
			log.Warningf("simplefs: filewalker for query failed: %s", err)
		}

		close(files)

	}()

	return dsq.ResultsFromIterator(q, dsq.Iterator{
		Next: func() (dsq.Result, bool) {
			select {
			case entry := <-files:
				if entry == nil {
					return dsq.Result{}, false
				}
				// log.Tracef("processing: %s", entry.Key)
				if !q.KeysOnly {
					objPath, err := d.buildPath(entry.Key)
					if err != nil {
						return dsq.Result{Error: err}, false
					}
					data, err := ioutil.ReadFile(objPath)
					if err != nil {
						return dsq.Result{Error: fmt.Errorf("error reading file %s: %s", entry.Key, err)}, false
					}
					newKey := ds.RawKey(entry.Key)
					wrapper, err := dbutils.NewWrapper(&newKey, data)
					if err != nil {
						return dsq.Result{Error: fmt.Errorf("failed to create wrapper for %s: %s", entry.Key, err)}, false
					}
					entry.Value = wrapper
				}
				return dsq.Result{Entry: *entry, Error: nil}, true
			case <-time.After(10 * time.Second):
				return dsq.Result{Error: errors.New("filesystem timeout")}, false
			}
		},
		Close: func() error {
			if stopWalkingFlag.SetToIf(false, true) {
				close(stopWalking)
			}
			return nil
		},
	}), nil
}

//
// func (d *datastore) Query(q dsq.Query) (dsq.Results, error) {
// 	return d.QueryNew(q)
// }
//
// func (d *datastore) QueryNew(q dsq.Query) (dsq.Results, error) {
// 	if len(q.Filters) > 0 ||
// 		len(q.Orders) > 0 ||
// 		q.Limit > 0 ||
// 		q.Offset > 0 {
// 		return d.QueryOrig(q)
// 	}
// 	var rnge *util.Range
// 	if q.Prefix != "" {
// 		rnge = util.BytesPrefix([]byte(q.Prefix))
// 	}
// 	i := d.DB.NewIterator(rnge, nil)
// 	return dsq.ResultsFromIterator(q, dsq.Iterator{
// 		Next: func() (dsq.Result, bool) {
// 			ok := i.Next()
// 			if !ok {
// 				return dsq.Result{}, false
// 			}
// 			k := string(i.Key())
// 			e := dsq.Entry{Key: k}
//
// 			if !q.KeysOnly {
// 				buf := make([]byte, len(i.Value()))
// 				copy(buf, i.Value())
// 				e.Value = buf
// 			}
// 			return dsq.Result{Entry: e}, true
// 		},
// 		Close: func() error {
// 			i.Release()
// 			return nil
// 		},
// 	}), nil
// }
//
// func (d *datastore) QueryOrig(q dsq.Query) (dsq.Results, error) {
// 	// we can use multiple iterators concurrently. see:
// 	// https://godoc.org/github.com/syndtr/goleveldb/leveldb#DB.NewIterator
// 	// advance the iterator only if the reader reads
// 	//
// 	// run query in own sub-process tied to Results.Process(), so that
// 	// it waits for us to finish AND so that clients can signal to us
// 	// that resources should be reclaimed.
// 	qrb := dsq.NewResultBuilder(q)
// 	qrb.Process.Go(func(worker goprocess.Process) {
// 		d.runQuery(worker, qrb)
// 	})
//
// 	// go wait on the worker (without signaling close)
// 	go qrb.Process.CloseAfterChildren()
//
// 	// Now, apply remaining things (filters, order)
// 	qr := qrb.Results()
// 	for _, f := range q.Filters {
// 		qr = dsq.NaiveFilter(qr, f)
// 	}
// 	for _, o := range q.Orders {
// 		qr = dsq.NaiveOrder(qr, o)
// 	}
// 	return qr, nil
// }
//
// func (d *datastore) runQuery(worker goprocess.Process, qrb *dsq.ResultBuilder) {
//
// 	var rnge *util.Range
// 	if qrb.Query.Prefix != "" {
// 		rnge = util.BytesPrefix([]byte(qrb.Query.Prefix))
// 	}
// 	i := d.DB.NewIterator(rnge, nil)
// 	defer i.Release()
//
// 	// advance iterator for offset
// 	if qrb.Query.Offset > 0 {
// 		for j := 0; j < qrb.Query.Offset; j++ {
// 			i.Next()
// 		}
// 	}
//
// 	// iterate, and handle limit, too
// 	for sent := 0; i.Next(); sent++ {
// 		// end early if we hit the limit
// 		if qrb.Query.Limit > 0 && sent >= qrb.Query.Limit {
// 			break
// 		}
//
// 		k := string(i.Key())
// 		e := dsq.Entry{Key: k}
//
// 		if !qrb.Query.KeysOnly {
// 			buf := make([]byte, len(i.Value()))
// 			copy(buf, i.Value())
// 			e.Value = buf
// 		}
//
// 		select {
// 		case qrb.Output <- dsq.Result{Entry: e}: // we sent it out
// 		case <-worker.Closing(): // client told us to end early.
// 			break
// 		}
// 	}
//
// 	if err := i.Error(); err != nil {
// 		select {
// 		case qrb.Output <- dsq.Result{Error: err}: // client read our error
// 		case <-worker.Closing(): // client told us to end.
// 			return
// 		}
// 	}
// }

func (d *datastore) Close() (err error) {
	return nil
}

func (d *datastore) IsThreadSafe() {}
