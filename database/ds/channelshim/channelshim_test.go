// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package channelshim

import (
	"io"
	"sync"
	"testing"

	datastore "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

var cds datastore.Datastore
var key datastore.Key
var q query.Query
var wg sync.WaitGroup

func testFunctions(testClose bool) {
	wg.Add(1)
	defer wg.Done()

	cds.Put(key, "value")
	cds.Get(key)
	cds.Has(key)
	cds.Delete(key)
	cds.Query(q)
	if batchingDS, ok := cds.(datastore.Batching); ok {
		batchingDS.Batch()
	}

	if testClose {
		if closingDS, ok := cds.(io.Closer); ok {
			closingDS.Close()
		}
	}

}

func TestChanneledDatastore(t *testing.T) {

	cds = NewChanneledDatastore(datastore.NewNullDatastore())
	key = datastore.RandomKey()

	// test normal concurrency-safe operation
	for i := 0; i < 100; i++ {
		go testFunctions(false)
	}
	wg.Wait()

	// test shutdown procedure
	for i := 0; i < 50; i++ {
		go testFunctions(false)
	}
	for i := 0; i < 50; i++ {
		go testFunctions(true)
	}
	wg.Wait()

	// test closed functions, just to be sure
	go testFunctions(true)
	wg.Wait()

}
