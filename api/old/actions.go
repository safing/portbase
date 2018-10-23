// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package api

import (
	"fmt"

	_ "github.com/Safing/safing-core/configuration"
	"github.com/Safing/safing-core/database"
	"github.com/Safing/safing-core/formats/dsd"

	"github.com/ipfs/go-datastore"
)

func Get(session *Session, key string) {
	iterator, err := database.EasyQueryIterator(key)
	if err != nil {
		handleError(session, fmt.Sprintf("error|500|could not query: %s", err))
		return
	}

	var returnedStuff bool

	for obj, ok := iterator.NextSync(); ok; obj, ok = iterator.NextSync() {
		bytes, err := database.DumpModel(obj.Value, dsd.JSON)

		returnedStuff = true

		if err == nil {
			toSend := []byte(fmt.Sprintf("current|%s|%s", obj.Key, string(bytes)))
			session.send <- toSend
		} else {
			handleError(session, fmt.Sprintf("error|500|dump failed: %s", err))
		}
	}

	if !returnedStuff {
		handleError(session, "error|400|no results: "+key)
	}
}

func Subscribe(session *Session, key string) {
	session.Subscribe(key)
	Get(session, key)
}

func Unsubscribe(session *Session, key string) {
	session.Unsubscribe(key)
}

func Save(session *Session, key string, create bool, data []byte) {
	var model database.Model
	var err error
	dbKey := datastore.NewKey(key)
	model, err = database.NewWrapper(&dbKey, data)
	if err != nil {
		handleError(session, fmt.Sprintf("error|500|failed to wrap object: %s", err))
		return
	}
	if create {
		err = database.Create(dbKey, model)
	} else {
		err = database.Update(dbKey, model)
	}
	if err != nil {
		handleError(session, fmt.Sprintf("error|500|failed to save to database: %s", err))
	}
}

func Delete(session *Session, key string) {
	dbKey := datastore.NewKey(key)
	err := database.Delete(dbKey)
	if err != nil {
		handleError(session, fmt.Sprintf("error|500|failed to delete from database: %s", err))
	}
}
