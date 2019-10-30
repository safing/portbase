//nolint:unparam,maligned
package hashmap

import (
	"reflect"
	"sync"
	"testing"

	"github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"
)

type TestRecord struct {
	record.Base
	sync.Mutex
	S    string
	I    int
	I8   int8
	I16  int16
	I32  int32
	I64  int64
	UI   uint
	UI8  uint8
	UI16 uint16
	UI32 uint32
	UI64 uint64
	F32  float32
	F64  float64
	B    bool
}

func TestHashMap(t *testing.T) {
	// start
	db, err := NewHashMap("test", "")
	if err != nil {
		t.Fatal(err)
	}

	a := &TestRecord{
		S:    "banana",
		I:    42,
		I8:   42,
		I16:  42,
		I32:  42,
		I64:  42,
		UI:   42,
		UI8:  42,
		UI16: 42,
		UI32: 42,
		UI64: 42,
		F32:  42.42,
		F64:  42.42,
		B:    true,
	}
	a.SetMeta(&record.Meta{})
	a.Meta().Update()
	a.SetKey("test:A")

	// put record
	err = db.Put(a)
	if err != nil {
		t.Fatal(err)
	}

	// get and compare
	a1, err := db.Get("A")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(a, a1) {
		t.Fatalf("mismatch, got %v", a1)
	}

	// setup query test records
	qA := &TestRecord{}
	qA.SetKey("test:path/to/A")
	qA.CreateMeta()
	qB := &TestRecord{}
	qB.SetKey("test:path/to/B")
	qB.CreateMeta()
	qC := &TestRecord{}
	qC.SetKey("test:path/to/C")
	qC.CreateMeta()
	qZ := &TestRecord{}
	qZ.SetKey("test:z")
	qZ.CreateMeta()
	// put
	err = db.Put(qA)
	if err == nil {
		err = db.Put(qB)
	}
	if err == nil {
		err = db.Put(qC)
	}
	if err == nil {
		err = db.Put(qZ)
	}
	if err != nil {
		t.Fatal(err)
	}

	// test query
	q := query.New("test:path/to/").MustBeValid()
	it, err := db.Query(q, true, true)
	if err != nil {
		t.Fatal(err)
	}
	cnt := 0
	for range it.Next {
		cnt++
	}
	if it.Err() != nil {
		t.Fatal(it.Err())
	}
	if cnt != 3 {
		t.Fatalf("unexpected query result count: %d", cnt)
	}

	// delete
	err = db.Delete("A")
	if err != nil {
		t.Fatal(err)
	}

	// check if its gone
	_, err = db.Get("A")
	if err == nil {
		t.Fatal("should fail")
	}

	// maintenance
	err = db.Maintain()
	if err != nil {
		t.Fatal(err)
	}
	err = db.MaintainThorough()
	if err != nil {
		t.Fatal(err)
	}

	// shutdown
	err = db.Shutdown()
	if err != nil {
		t.Fatal(err)
	}
}
