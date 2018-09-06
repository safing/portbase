// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import (
	"errors"
)

// Errors
var (
	ErrNotFound         = errors.New("database: entry could not be found")
	ErrPermissionDenied = errors.New("database: access to record denied")
)

func init() {
	// if strings.HasSuffix(os.Args[0], ".test") {
	// 	// testing setup
	// 	log.Warning("===== DATABASE RUNNING IN TEST MODE =====")
	// 	db = channelshim.NewChanneledDatastore(ds.NewMapDatastore())
	// 	return
	// }

	// sfsDB, err := simplefs.NewDatastore(meta.DatabaseDir())
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "FATAL ERROR: could not init simplefs database: %s\n", err)
	// 	os.Exit(1)
	// }

	// ldb, err := leveldb.NewDatastore(path.Join(meta.DatabaseDir(), "leveldb"), &leveldb.Options{})
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "FATAL ERROR: could not init simplefs database: %s\n", err)
	// 	os.Exit(1)
	// }
	//
	// mapDB := ds.NewMapDatastore()
	//
	// db = channelshim.NewChanneledDatastore(mount.New([]mount.Mount{
	// 	mount.Mount{
	// 		Prefix:    ds.NewKey("/Run"),
	// 		Datastore: mapDB,
	// 	},
	// 	mount.Mount{
	// 		Prefix:    ds.NewKey("/"),
	// 		Datastore: ldb,
	// 	},
	// }))

}

// func Batch() (ds.Batch, error) {
//   return db.Batch()
// }

// func Close() error {
//   return db.Close()
// }

// func Get(key *ds.Key) (Model, error) {
// 	data, err := db.Get(*key)
// 	if err != nil {
// 		switch err {
// 		case ds.ErrNotFound:
// 			return nil, ErrNotFound
// 		default:
// 			return nil, err
// 		}
// 	}
// 	model, ok := data.(Model)
// 	if !ok {
// 		return nil, errors.New("database did not return model")
// 	}
// 	return model, nil
// }

// func Has(key ds.Key) (exists bool, err error) {
// 	return db.Has(key)
// }
//
// func Create(key ds.Key, model Model) (err error) {
// 	handleCreateSubscriptions(model)
// 	err = db.Put(key, model)
// 	if err != nil {
// 		log.Tracef("database: failed to create entry %s: %s", key, err)
// 	}
// 	return err
// }
//
// func Update(key ds.Key, model Model) (err error) {
// 	handleUpdateSubscriptions(model)
// 	err = db.Put(key, model)
// 	if err != nil {
// 		log.Tracef("database: failed to update entry %s: %s", key, err)
// 	}
// 	return err
// }
//
// func Delete(key ds.Key) (err error) {
// 	handleDeleteSubscriptions(&key)
// 	return db.Delete(key)
// }
//
// func Query(q dsq.Query) (dsq.Results, error) {
// 	return db.Query(q)
// }
//
// func RawGet(key ds.Key) (*dbutils.Wrapper, error) {
// 	data, err := db.Get(key)
// 	if err != nil {
// 		return nil, err
// 	}
// 	wrapped, ok := data.(*dbutils.Wrapper)
// 	if !ok {
// 		return nil, errors.New("returned data is not a wrapper")
// 	}
// 	return wrapped, nil
// }
//
// func RawPut(key ds.Key, value interface{}) error {
// 	return db.Put(key, value)
// }
