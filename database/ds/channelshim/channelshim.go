// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package channelshim

import (
	"errors"
	"io"
	"time"

	datastore "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	"github.com/tevino/abool"
)

var ErrDatastoreClosed = errors.New("datastore: this instance was closed")

// ChanneledDatastore makes datastore thread-safe
type ChanneledDatastore struct {
	child    datastore.Datastore
	setChild chan datastore.Datastore

	putRequest    chan *VKeyValue
	putReply      chan *error
	getRequest    chan *datastore.Key
	getReply      chan *VValueErr
	hasRequest    chan *datastore.Key
	hasReply      chan *VExistsErr
	deleteRequest chan *datastore.Key
	deleteReply   chan *error
	queryRequest  chan *dsq.Query
	queryReply    chan *VResultsErr
	batchRequest  chan interface{} // nothing actually
	batchReply    chan *VBatchErr
	closeRequest  chan interface{} // nothing actually
	closeReply    chan *error

	closedFlag *abool.AtomicBool
}

type VKeyValue struct {
	Key   datastore.Key
	Value interface{}
}

type VValueErr struct {
	Value interface{}
	Err   error
}

type VExistsErr struct {
	Exists bool
	Err    error
}

type VResultsErr struct {
	Results dsq.Results
	Err     error
}

type VBatchErr struct {
	Batch datastore.Batch
	Err   error
}

func (cds *ChanneledDatastore) run() {
	if cds.child == nil {
		cds.child = <-cds.setChild
	}
	for {
		select {
		case v := <-cds.putRequest:
			cds.put(v)
		case v := <-cds.getRequest:
			cds.get(v)
		case v := <-cds.hasRequest:
			cds.has(v)
		case v := <-cds.deleteRequest:
			cds.delete(v)
		case v := <-cds.queryRequest:
			cds.query(v)
		case <-cds.batchRequest:
			cds.batch()
		case <-cds.closeRequest:
			err := cds.close()
			if err == nil {
				cds.closeReply <- &err
				defer cds.stop()
				return
			}
			cds.closeReply <- &err
		}
	}
}

func (cds *ChanneledDatastore) stop() {
	for {
		select {
		case <-cds.putRequest:
			cds.putReply <- &ErrDatastoreClosed
		case <-cds.getRequest:
			cds.getReply <- &VValueErr{nil, ErrDatastoreClosed}
		case <-cds.hasRequest:
			cds.hasReply <- &VExistsErr{false, ErrDatastoreClosed}
		case <-cds.deleteRequest:
			cds.deleteReply <- &ErrDatastoreClosed
		case <-cds.queryRequest:
			cds.queryReply <- &VResultsErr{nil, ErrDatastoreClosed}
		case <-cds.batchRequest:
			cds.batchReply <- &VBatchErr{nil, ErrDatastoreClosed}
		case <-cds.closeRequest:
			cds.closeReply <- &ErrDatastoreClosed
		case <-time.After(1 * time.Minute):
			// TODO: theoretically a race condition, as some goroutines _could_ still be stuck in front of the request channel
			close(cds.putRequest)
			close(cds.putReply)
			close(cds.getRequest)
			close(cds.getReply)
			close(cds.hasRequest)
			close(cds.hasReply)
			close(cds.deleteRequest)
			close(cds.deleteReply)
			close(cds.queryRequest)
			close(cds.queryReply)
			close(cds.batchRequest)
			close(cds.batchReply)
			close(cds.closeRequest)
			close(cds.closeReply)
			return
		}
	}
}

// NewChanneledDatastore constructs a datastore accessed through channels.
func NewChanneledDatastore(ds datastore.Datastore) *ChanneledDatastore {
	cds := &ChanneledDatastore{child: ds}
	cds.setChild = make(chan datastore.Datastore)

	cds.putRequest = make(chan *VKeyValue)
	cds.putReply = make(chan *error)
	cds.getRequest = make(chan *datastore.Key)
	cds.getReply = make(chan *VValueErr)
	cds.hasRequest = make(chan *datastore.Key)
	cds.hasReply = make(chan *VExistsErr)
	cds.deleteRequest = make(chan *datastore.Key)
	cds.deleteReply = make(chan *error)
	cds.queryRequest = make(chan *dsq.Query)
	cds.queryReply = make(chan *VResultsErr)
	cds.batchRequest = make(chan interface{})
	cds.batchReply = make(chan *VBatchErr)
	cds.closeRequest = make(chan interface{})
	cds.closeReply = make(chan *error)

	cds.closedFlag = abool.NewBool(false)

	go cds.run()

	return cds
}

// SetChild sets the child of the datastore, if not yet set
func (cds *ChanneledDatastore) SetChild(ds datastore.Datastore) error {
	select {
	case cds.setChild <- ds:
	default:
		return errors.New("channelshim: child already set")
	}
	return nil
}

// Children implements Shim
func (cds *ChanneledDatastore) Children() []datastore.Datastore {
	return []datastore.Datastore{cds.child}
}

// Put implements Datastore.Put
func (cds *ChanneledDatastore) Put(key datastore.Key, value interface{}) error {
	if cds.closedFlag.IsSet() {
		return ErrDatastoreClosed
	}
	cds.putRequest <- &VKeyValue{key, value}
	err := <-cds.putReply
	return *err
}

func (cds *ChanneledDatastore) put(v *VKeyValue) {
	err := cds.child.Put(v.Key, v.Value)
	cds.putReply <- &err
}

// Get implements Datastore.Get
func (cds *ChanneledDatastore) Get(key datastore.Key) (value interface{}, err error) {
	if cds.closedFlag.IsSet() {
		return nil, ErrDatastoreClosed
	}
	cds.getRequest <- &key
	v := <-cds.getReply
	return v.Value, v.Err
}

func (cds *ChanneledDatastore) get(key *datastore.Key) {
	value, err := cds.child.Get(*key)
	cds.getReply <- &VValueErr{value, err}
}

// Has implements Datastore.Has
func (cds *ChanneledDatastore) Has(key datastore.Key) (exists bool, err error) {
	if cds.closedFlag.IsSet() {
		return false, ErrDatastoreClosed
	}
	cds.hasRequest <- &key
	v := <-cds.hasReply
	return v.Exists, v.Err
}

func (cds *ChanneledDatastore) has(key *datastore.Key) {
	exists, err := cds.child.Has(*key)
	cds.hasReply <- &VExistsErr{exists, err}
}

// Delete implements Datastore.Delete
func (cds *ChanneledDatastore) Delete(key datastore.Key) error {
	if cds.closedFlag.IsSet() {
		return ErrDatastoreClosed
	}
	cds.deleteRequest <- &key
	err := <-cds.deleteReply
	return *err
}

func (cds *ChanneledDatastore) delete(key *datastore.Key) {
	err := cds.child.Delete(*key)
	cds.deleteReply <- &err
}

// Query implements Datastore.Query
func (cds *ChanneledDatastore) Query(q dsq.Query) (dsq.Results, error) {
	if cds.closedFlag.IsSet() {
		return nil, ErrDatastoreClosed
	}
	cds.queryRequest <- &q
	v := <-cds.queryReply
	return v.Results, v.Err
}

func (cds *ChanneledDatastore) query(q *dsq.Query) {
	results, err := cds.child.Query(*q)
	cds.queryReply <- &VResultsErr{results, err}
}

// Query implements Datastore.Batch
func (cds *ChanneledDatastore) Batch() (datastore.Batch, error) {
	if cds.closedFlag.IsSet() {
		return nil, ErrDatastoreClosed
	}
	cds.batchRequest <- nil
	v := <-cds.batchReply
	return v.Batch, v.Err
}

func (cds *ChanneledDatastore) batch() {
	if bds, ok := cds.child.(datastore.Batching); ok {
		batch, err := bds.Batch()
		cds.batchReply <- &VBatchErr{batch, err}
	} else {
		cds.batchReply <- &VBatchErr{nil, datastore.ErrBatchUnsupported}
	}
}

// Query closed child Datastore and this Shim
func (cds *ChanneledDatastore) Close() error {
	if cds.closedFlag.IsSet() {
		return ErrDatastoreClosed
	}
	cds.closeRequest <- nil
	err := <-cds.closeReply
	return *err
}

func (cds *ChanneledDatastore) close() error {
	if cds, ok := cds.child.(io.Closer); ok {
		return cds.Close()
	}
	return nil
}

func (cds *ChanneledDatastore) IsThreadSafe() {

}
