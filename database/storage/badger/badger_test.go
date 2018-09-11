package badger

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/Safing/portbase/database/record"
)

type TestRecord struct {
	record.Base
	lock sync.Mutex
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

func (tr *TestRecord) Lock() {
}

func (tr *TestRecord) Unlock() {
}

func TestBadger(t *testing.T) {
	testDir, err := ioutil.TempDir("", "testing-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir) // clean up

	db, err := NewBadger("test", testDir)
	if err != nil {
		t.Fatal(err)
	}

	a := &TestRecord{S: "banana"}
	a.SetMeta(&record.Meta{})
	a.Meta().Update()
	a.SetKey("test:A")

	err = db.Put(a)
	if err != nil {
		t.Fatal(err)
	}

	r1, err := db.Get("A")
	if err != nil {
		t.Fatal(err)
	}

	a1 := r1.(*TestRecord)

	if a.S != a1.S {
		t.Fatal("mismatch")
	}
}
